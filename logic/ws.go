package logic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"tgwp/log/zlog"
	"tgwp/response"
	"tgwp/types"
)

const (
	wsPingInterval = 30 * time.Second
	wsPongWait     = 40 * time.Second
	wsWriteWait    = 10 * time.Second
	wsReadLimit    = 64 * 1024
)

type WsHandler func(ctx *WsContext, data json.RawMessage) error

type WsContext struct {
	Ctx    context.Context
	Conn   *websocket.Conn
	UserID int64
	RootID int64
	Hub    *WsHub
}

type wsConnInfo struct {
	UserID  int64
	RootID  int64
	WriteMu sync.Mutex
}

type WsHub struct {
	upgrader  websocket.Upgrader
	mu        sync.RWMutex
	connInfo  map[*websocket.Conn]*wsConnInfo
	userConns map[int64]map[*websocket.Conn]struct{}
	roomConns map[int64]map[*websocket.Conn]struct{}
	handlers  map[string]WsHandler
}

var wsHubOnce sync.Once
var wsHub *WsHub

func GetWsHub() *WsHub {
	wsHubOnce.Do(func() {
		wsHub = NewWsHub()
	})
	return wsHub
}

func NewWsHub() *WsHub {
	hub := &WsHub{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connInfo:  make(map[*websocket.Conn]*wsConnInfo),
		userConns: make(map[int64]map[*websocket.Conn]struct{}),
		roomConns: make(map[int64]map[*websocket.Conn]struct{}),
		handlers:  make(map[string]WsHandler),
	}
	hub.RegisterHandler("ping", hub.handlePing)
	hub.RegisterHandler("team_room_join", hub.handleTeamRoomJoin)
	hub.RegisterHandler("team_room_leave", hub.handleTeamRoomLeave)
	return hub
}

func (h *WsHub) RegisterHandler(msgType string, handler WsHandler) {
	h.handlers[msgType] = handler
}

func (h *WsHub) Serve(ctx context.Context, w http.ResponseWriter, r *http.Request, userID int64, rootID int64) error {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	h.register(conn, userID, rootID)
	if userID > 0 && rootID > 0 {
		if err := h.autoJoinTeamRoom(ctx, conn, userID, rootID); err != nil {
			zlog.CtxWarnf(ctx, "websocket自动加入团队房间失败:%v", err)
		}
	}
	go h.readLoop(ctx, conn)
	go h.heartbeatLoop(ctx, conn)
	return nil
}

func (h *WsHub) register(conn *websocket.Conn, userID int64, rootID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connInfo[conn] = &wsConnInfo{UserID: userID, RootID: rootID}
	if _, ok := h.userConns[userID]; !ok {
		h.userConns[userID] = make(map[*websocket.Conn]struct{})
	}
	h.userConns[userID][conn] = struct{}{}
	if rootID > 0 {
		if _, ok := h.roomConns[rootID]; !ok {
			h.roomConns[rootID] = make(map[*websocket.Conn]struct{})
		}
		h.roomConns[rootID][conn] = struct{}{}
	}
}

func (h *WsHub) BindRoom(conn *websocket.Conn, rootID int64) {
	if rootID <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	info, ok := h.connInfo[conn]
	if !ok {
		return
	}
	if info.RootID == rootID {
		return
	}
	if info.RootID > 0 {
		if roomSet, ok := h.roomConns[info.RootID]; ok {
			delete(roomSet, conn)
			if len(roomSet) == 0 {
				delete(h.roomConns, info.RootID)
			}
		}
	}
	info.RootID = rootID
	if _, ok := h.roomConns[rootID]; !ok {
		h.roomConns[rootID] = make(map[*websocket.Conn]struct{})
	}
	h.roomConns[rootID][conn] = struct{}{}
}

func (h *WsHub) UnbindRoom(conn *websocket.Conn, rootID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	info, ok := h.connInfo[conn]
	if !ok {
		return
	}
	if rootID <= 0 {
		rootID = info.RootID
	}
	if rootID <= 0 {
		return
	}
	if roomSet, ok := h.roomConns[rootID]; ok {
		delete(roomSet, conn)
		if len(roomSet) == 0 {
			delete(h.roomConns, rootID)
		}
	}
	if info.RootID == rootID {
		info.RootID = 0
	}
}

func (h *WsHub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	info, ok := h.connInfo[conn]
	if !ok {
		h.mu.Unlock()
		return
	}
	userID := info.UserID
	rootID := info.RootID
	delete(h.connInfo, conn)
	if userSet, ok := h.userConns[info.UserID]; ok {
		delete(userSet, conn)
		if len(userSet) == 0 {
			delete(h.userConns, info.UserID)
		}
	}
	if info.RootID > 0 {
		if roomSet, ok := h.roomConns[info.RootID]; ok {
			delete(roomSet, conn)
			if len(roomSet) == 0 {
				delete(h.roomConns, info.RootID)
			}
		}
	}
	h.mu.Unlock()
	if userID > 0 && rootID > 0 {
		if err := h.autoLeaveTeamRoom(context.Background(), conn, userID, rootID); err != nil {
			zlog.Warnf("websocket自动离开团队房间失败:%v", err)
		}
	}
	_ = conn.Close()
}

func (h *WsHub) getInfo(conn *websocket.Conn) (*wsConnInfo, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	info, ok := h.connInfo[conn]
	return info, ok
}

func (h *WsHub) readLoop(ctx context.Context, conn *websocket.Conn) {
	conn.SetReadLimit(wsReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			h.unregister(conn)
			return
		}
		h.handleMessage(ctx, conn, message)
	}
}

func (h *WsHub) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := h.writePing(conn); err != nil {
			zlog.CtxWarnf(ctx, "websocket心跳失败:%v", err)
			h.unregister(conn)
			return
		}
	}
}

func (h *WsHub) handleMessage(ctx context.Context, conn *websocket.Conn, message []byte) {
	var req types.WsRequest
	if err := json.Unmarshal(message, &req); err != nil || req.Type == "" {
		_ = h.Send(conn, types.WsResponse{
			Type:    "error",
			Code:    response.PARAM_NOT_VALID.Code,
			Message: "消息格式错误",
		})
		return
	}
	handler, ok := h.handlers[req.Type]
	if !ok {
		_ = h.Send(conn, types.WsResponse{
			Type:    "error",
			Code:    response.MESSAGE_NOT_EXIST.Code,
			Message: "消息类型不存在",
		})
		return
	}
	info, ok := h.getInfo(conn)
	if !ok {
		return
	}
	err := handler(&WsContext{
		Ctx:    ctx,
		Conn:   conn,
		UserID: info.UserID,
		RootID: info.RootID,
		Hub:    h,
	}, req.Data)
	if err != nil {
		_ = h.Send(conn, types.WsResponse{
			Type:    "error",
			Code:    response.COMMON_FAIL.Code,
			Message: err.Error(),
		})
	}
}

func (h *WsHub) handlePing(ctx *WsContext, data json.RawMessage) error {
	return h.Send(ctx.Conn, types.WsResponse{
		Type:    "pong",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]int64{
			"ts": time.Now().Unix(),
		},
	})
}

func (h *WsHub) handleTeamRoomJoin(ctx *WsContext, data json.RawMessage) error {
	var req types.TeamRoomWsJoinReq
	if err := json.Unmarshal(data, &req); err != nil {
		return errors.New("param blank")
	}
	roomIDStr := req.RoomID
	if roomIDStr == "" && ctx.RootID > 0 {
		roomIDStr = strconv.FormatInt(ctx.RootID, 10)
	}
	roomID, err := parseTeamRoomID(roomIDStr)
	if err != nil {
		return errors.New("param blank")
	}
	roomInfo, err := NewTeamRoomLogic().JoinRoom(ctx.Ctx, ctx.UserID, roomID)
	if err != nil {
		return err
	}
	h.BindRoom(ctx.Conn, roomID)
	h.SendToRoom(roomID, types.WsResponse{
		Type:    "team_room_member_update",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":    roomInfo,
			"action":  "join",
			"user_id": ctx.UserID,
		},
	})
	return nil
}

func (h *WsHub) handleTeamRoomLeave(ctx *WsContext, data json.RawMessage) error {
	var req types.TeamRoomWsLeaveReq
	if err := json.Unmarshal(data, &req); err != nil {
		return errors.New("param blank")
	}
	roomIDStr := req.RoomID
	if roomIDStr == "" && ctx.RootID > 0 {
		roomIDStr = strconv.FormatInt(ctx.RootID, 10)
	}
	roomID, err := parseTeamRoomID(roomIDStr)
	if err != nil {
		return errors.New("param blank")
	}
	roomInfo, err := NewTeamRoomLogic().LeaveRoom(ctx.Ctx, ctx.UserID, roomID)
	if err != nil {
		return err
	}
	h.SendToRoom(roomID, types.WsResponse{
		Type:    "team_room_member_update",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":    roomInfo,
			"action":  "leave",
			"user_id": ctx.UserID,
		},
	})
	h.UnbindRoom(ctx.Conn, roomID)
	return nil
}

func (h *WsHub) autoJoinTeamRoom(ctx context.Context, conn *websocket.Conn, userID int64, roomID int64) error {
	roomInfo, err := NewTeamRoomLogic().JoinRoom(ctx, userID, roomID)
	if err != nil {
		return err
	}
	h.BindRoom(conn, roomID)
	h.SendToRoom(roomID, types.WsResponse{
		Type:    "team_room_member_update",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":    roomInfo,
			"action":  "join",
			"user_id": userID,
		},
	})
	return nil
}

func (h *WsHub) autoLeaveTeamRoom(ctx context.Context, conn *websocket.Conn, userID int64, roomID int64) error {
	roomInfo, err := NewTeamRoomLogic().LeaveRoom(ctx, userID, roomID)
	if err != nil {
		return err
	}
	h.UnbindRoom(conn, roomID)
	h.SendToRoom(roomID, types.WsResponse{
		Type:    "team_room_member_update",
		Code:    response.SUCCESS.Code,
		Message: response.SUCCESS.Msg,
		Data: map[string]interface{}{
			"room":    roomInfo,
			"action":  "leave",
			"user_id": userID,
		},
	})
	return nil
}

func (h *WsHub) Send(conn *websocket.Conn, resp types.WsResponse) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return h.writeMessage(conn, websocket.TextMessage, payload)
}

func (h *WsHub) SendToUser(userID int64, resp types.WsResponse) {
	conns := h.getUserConnections(userID)
	for _, conn := range conns {
		if err := h.Send(conn, resp); err != nil {
			h.unregister(conn)
		}
	}
}

func (h *WsHub) SendToRoom(rootID int64, resp types.WsResponse) {
	conns := h.getRoomConnections(rootID)
	for _, conn := range conns {
		if err := h.Send(conn, resp); err != nil {
			h.unregister(conn)
		}
	}
}

func (h *WsHub) getUserConnections(userID int64) []*websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	set := h.userConns[userID]
	if len(set) == 0 {
		return nil
	}
	conns := make([]*websocket.Conn, 0, len(set))
	for conn := range set {
		conns = append(conns, conn)
	}
	return conns
}

func (h *WsHub) getRoomConnections(rootID int64) []*websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	set := h.roomConns[rootID]
	if len(set) == 0 {
		return nil
	}
	conns := make([]*websocket.Conn, 0, len(set))
	for conn := range set {
		conns = append(conns, conn)
	}
	return conns
}

func (h *WsHub) ActiveUserIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.userConns) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(h.userConns))
	for userID := range h.userConns {
		ids = append(ids, userID)
	}
	return ids
}

func (h *WsHub) ActiveRoomUserIDs(rootID int64) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	set := h.roomConns[rootID]
	if len(set) == 0 {
		return nil
	}
	unique := make(map[int64]struct{})
	for conn := range set {
		info, ok := h.connInfo[conn]
		if !ok {
			continue
		}
		if info.UserID == 0 {
			continue
		}
		unique[info.UserID] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	return ids
}

func (h *WsHub) writePing(conn *websocket.Conn) error {
	info, ok := h.getInfo(conn)
	if !ok {
		return errors.New("连接不存在")
	}
	info.WriteMu.Lock()
	defer info.WriteMu.Unlock()
	return conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(wsWriteWait))
}

func (h *WsHub) writeMessage(conn *websocket.Conn, messageType int, payload []byte) error {
	info, ok := h.getInfo(conn)
	if !ok {
		return errors.New("连接不存在")
	}
	info.WriteMu.Lock()
	defer info.WriteMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return conn.WriteMessage(messageType, payload)
}
