package types

import "time"

type TeamRoomCreateReq struct {
	Mode string `json:"mode" form:"mode"`
}

type TeamRoomCreateResp struct {
	Room TeamRoomInfo `json:"room"`
}

type TeamRoomInfoReq struct {
	RoomID string `form:"room_id" json:"room_id"`
}

type TeamRoomInfoResp struct {
	Room TeamRoomInfo `json:"room"`
}

type TeamRoomListReq struct {
	Page   int   `form:"page" json:"page"`
	Limit  int   `form:"limit" json:"limit"`
	Status *int8 `form:"status" json:"status"`
}

type TeamRoomListResp struct {
	Total int64              `json:"total"`
	Rooms []TeamRoomListItem `json:"rooms"`
}

type TeamRoomModeListResp struct {
	Modes []TeamRoomModeInfo `json:"modes"`
}

type TeamRoomModeInfo struct {
	Mode     string `json:"mode"`
	Duration int64  `json:"duration"`
	Problems []int  `json:"problems"`
}

type TeamRoomListItem struct {
	RoomID       int64     `json:"room_id,string"`
	Mode         string    `json:"mode"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	EndTime      int64     `json:"end_time"`
	PlayerCount  int       `json:"player_count"`
	ProblemCount int       `json:"problem_count"`
}

type TeamRoomInfo struct {
	RoomID      int64                    `json:"room_id,string"`
	Mode        string                   `json:"mode"`
	Status      int8                     `json:"status"`
	CreatedAt   time.Time                `json:"created_at"`
	EndTime     int64                    `json:"end_time"`
	Players     []TeamRoomPlayerInfo     `json:"players"`
	Problems    []TeamRoomProblemInfo    `json:"problems"`
	Submissions []TeamRoomSubmissionInfo `json:"submissions"`
	Score       int64                    `json:"score"`
	Duration    int64                    `json:"duration"`
}

type TeamRoomPlayerInfo struct {
	UserID   int64  `json:"user_id,string"`
	Username string `json:"username"`
	JoinAt   int64  `json:"join_at"`
}

type TeamRoomProblemInfo struct {
	ProblemID  string `json:"problem_id"`
	ProblemURL string `json:"problem_url"`
	Difficulty int    `json:"difficulty"`
	Solved     bool   `json:"solved"`
	SolvedBy   int64  `json:"solved_by,string"`
	Penalty    int    `json:"penalty"`
	SolvedAt   int64  `json:"solved_at"`
}

type TeamRoomSubmissionInfo struct {
	SubmissionID int64  `json:"submission_id,string"`
	ProblemID    string `json:"problem_id"`
	UserID       int64  `json:"user_id,string"`
	Verdict      string `json:"verdict"`
	SubmitTime   int64  `json:"submit_time"`
}

type TeamRoomWsJoinReq struct {
	RoomID string `json:"room_id"`
}

type TeamRoomWsLeaveReq struct {
	RoomID string `json:"room_id"`
}

type TeamRoomWsChatReq struct {
	RoomID  string `json:"room_id"`
	Content string `json:"content"`
}
