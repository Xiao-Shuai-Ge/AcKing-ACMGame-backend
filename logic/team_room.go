package logic

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	"tgwp/global"
	"tgwp/model"
	"tgwp/repo"
	"tgwp/response"
	"tgwp/types"
)

type TeamRoomLogic struct {
}

func NewTeamRoomLogic() *TeamRoomLogic {
	return &TeamRoomLogic{}
}

func (l *TeamRoomLogic) CreateRoom(ctx context.Context, userID int64, req types.TeamRoomCreateReq) (resp types.TeamRoomCreateResp, err error) {
	if userID == 0 || req.Mode == "" {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	modeConfig, ok := getTeamRoomModeConfig(req.Mode)
	if !ok {
		return resp, response.ErrResp(errors.New("mode not exist"), response.PARAM_NOT_VALID)
	}
	user, err := repo.NewUserRepo(global.DB).GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MEMBER_NOT_EXIST)
		}
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	problems, err := l.buildProblems(ctx, modeConfig.Problems)
	if err != nil {
		return resp, err
	}
	players := []teamRoomPlayer{{
		UserID:   user.ID,
		Username: user.Username,
		JoinAt:   time.Now().Unix(),
	}}
	problemStatus := make([]teamRoomProblemStatus, 0, len(problems))
	for _, p := range problems {
		problemStatus = append(problemStatus, teamRoomProblemStatus{
			ProblemID: p.ProblemID,
		})
	}
	problemBytes, _ := json.Marshal(problems)
	playerBytes, _ := json.Marshal(players)
	statusBytes, _ := json.Marshal(problemStatus)
	submissionBytes, _ := json.Marshal([]teamRoomSubmissionRecord{})
	duration := getTeamRoomDuration(req.Mode)
	extraBytes, _ := json.Marshal(teamRoomExtraInfo{
		DurationSeconds: int64(duration.Seconds()),
	})
	room := model.TeamRoom{
		Mode:              req.Mode,
		ProblemList:       string(problemBytes),
		PlayerList:        string(playerBytes),
		CreatorID:         userID,
		Status:            0,
		SubmissionRecords: string(submissionBytes),
		ProblemStatus:     string(statusBytes),
		ExtraInfo:         string(extraBytes),
	}
	if err := repo.NewTeamRoomRepo(global.DB).Create(&room); err != nil {
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	GetTeamRoomManager().StartRoom(room)
	resp.Room = buildTeamRoomInfo(room)
	return resp, nil
}

func (l *TeamRoomLogic) GetRoomInfo(ctx context.Context, req types.TeamRoomInfoReq) (resp types.TeamRoomInfoResp, err error) {
	_ = ctx
	roomID, err := parseTeamRoomID(req.RoomID)
	if err != nil {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	room, err := repo.NewTeamRoomRepo(global.DB).GetByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	resp.Room = buildTeamRoomInfo(room)
	return resp, nil
}

func (l *TeamRoomLogic) ListRooms(ctx context.Context, req types.TeamRoomListReq) (resp types.TeamRoomListResp, err error) {
	_ = ctx
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	rooms, total, err := repo.NewTeamRoomRepo(global.DB).List(offset, limit, req.Status)
	if err != nil {
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	items := make([]types.TeamRoomListItem, 0, len(rooms))
	for _, room := range rooms {
		players := parseTeamRoomPlayers(room.PlayerList)
		problems := parseTeamRoomProblems(room.ProblemList)
		items = append(items, types.TeamRoomListItem{
			RoomID:       room.ID,
			Mode:         room.Mode,
			Status:       room.Status,
			CreatedAt:    room.CreatedAt,
			EndTime:      room.EndTime,
			PlayerCount:  len(players),
			ProblemCount: len(problems),
		})
	}
	resp.Total = total
	resp.Rooms = items
	return resp, nil
}

func (l *TeamRoomLogic) ListModes(ctx context.Context) (resp types.TeamRoomModeListResp, err error) {
	_ = ctx
	resp.Modes = buildTeamRoomModeInfos()
	return resp, nil
}

func (l *TeamRoomLogic) JoinRoom(ctx context.Context, userID int64, roomID int64) (types.TeamRoomInfo, error) {
	if userID == 0 || roomID == 0 {
		return types.TeamRoomInfo{}, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	room, err := repo.NewTeamRoomRepo(global.DB).GetByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return types.TeamRoomInfo{}, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return types.TeamRoomInfo{}, response.ErrResp(err, response.DATABASE_ERROR)
	}
	if room.Status != 0 {
		return types.TeamRoomInfo{}, response.ErrResp(errors.New("room finished"), response.PARAM_NOT_VALID)
	}
	user, err := repo.NewUserRepo(global.DB).GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return types.TeamRoomInfo{}, response.ErrResp(err, response.MEMBER_NOT_EXIST)
		}
		return types.TeamRoomInfo{}, response.ErrResp(err, response.DATABASE_ERROR)
	}
	players := parseTeamRoomPlayers(room.PlayerList)
	updated := false
	found := false
	for i := range players {
		if players[i].UserID == userID {
			found = true
			if players[i].Username != user.Username {
				players[i].Username = user.Username
				updated = true
			}
			break
		}
	}
	if !found {
		players = append(players, teamRoomPlayer{
			UserID:   userID,
			Username: user.Username,
			JoinAt:   time.Now().Unix(),
		})
		updated = true
	}
	if updated {
		bytes, _ := json.Marshal(players)
		room.PlayerList = string(bytes)
		if err := repo.NewTeamRoomRepo(global.DB).UpdatePlayerList(room.ID, room.PlayerList); err != nil {
			return types.TeamRoomInfo{}, response.ErrResp(err, response.DATABASE_ERROR)
		}
	}
	return buildTeamRoomInfo(room), nil
}

func (l *TeamRoomLogic) LeaveRoom(ctx context.Context, userID int64, roomID int64) (types.TeamRoomInfo, error) {
	if userID == 0 || roomID == 0 {
		return types.TeamRoomInfo{}, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	room, err := repo.NewTeamRoomRepo(global.DB).GetByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return types.TeamRoomInfo{}, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return types.TeamRoomInfo{}, response.ErrResp(err, response.DATABASE_ERROR)
	}
	players := parseTeamRoomPlayers(room.PlayerList)
	updated := false
	if len(players) > 0 {
		next := players[:0]
		for _, p := range players {
			if p.UserID == userID {
				updated = true
				continue
			}
			next = append(next, p)
		}
		players = next
	}
	if updated {
		bytes, _ := json.Marshal(players)
		room.PlayerList = string(bytes)
		if err := repo.NewTeamRoomRepo(global.DB).UpdatePlayerList(room.ID, room.PlayerList); err != nil {
			return types.TeamRoomInfo{}, response.ErrResp(err, response.DATABASE_ERROR)
		}
	}
	return buildTeamRoomInfo(room), nil
}

func (l *TeamRoomLogic) buildProblems(ctx context.Context, preset []int) ([]teamRoomProblem, error) {
	problemRepo := repo.NewCodeforcesProblemRepo(global.DB)
	problems := make([]teamRoomProblem, 0, len(preset))
	used := make(map[string]struct{})
	for _, target := range preset {
		minDifficulty := target - 100
		if minDifficulty < 0 {
			minDifficulty = 0
		}
		maxDifficulty := target + 100
		var picked model.CodeforcesProblem
		var err error
		for attempt := 0; attempt < 10; attempt++ {
			picked, err = problemRepo.GetRandomByDifficulty(minDifficulty, maxDifficulty)
			if err != nil {
				break
			}
			if _, ok := used[picked.ID]; ok {
				continue
			}
			break
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
			}
			return nil, response.ErrResp(err, response.DATABASE_ERROR)
		}
		if picked.ID == "" {
			return nil, response.ErrResp(errors.New("problem empty"), response.MESSAGE_NOT_EXIST)
		}
		used[picked.ID] = struct{}{}
		problems = append(problems, teamRoomProblem{
			ProblemID:  picked.ID,
			ProblemURL: picked.Url,
			Difficulty: picked.Difficulty,
		})
	}
	rand.Shuffle(len(problems), func(i, j int) {
		problems[i], problems[j] = problems[j], problems[i]
	})
	return problems, nil
}

func parseTeamRoomID(roomID string) (int64, error) {
	if roomID == "" {
		return 0, errors.New("param blank")
	}
	return strconv.ParseInt(roomID, 10, 64)
}

func buildTeamRoomInfo(room model.TeamRoom) types.TeamRoomInfo {
	problems := parseTeamRoomProblems(room.ProblemList)
	status := parseTeamRoomProblemStatus(room.ProblemStatus)
	players := parseTeamRoomPlayers(room.PlayerList)
	submissions := parseTeamRoomSubmissions(room.SubmissionRecords)
	extra := parseTeamRoomExtra(room.ExtraInfo)
	problemMap := make(map[string]teamRoomProblemStatus, len(status))
	for _, item := range status {
		problemMap[item.ProblemID] = item
	}
	problemInfos := make([]types.TeamRoomProblemInfo, 0, len(problems))
	for _, p := range problems {
		stat := problemMap[p.ProblemID]
		problemInfos = append(problemInfos, types.TeamRoomProblemInfo{
			ProblemID:  p.ProblemID,
			ProblemURL: p.ProblemURL,
			Difficulty: p.Difficulty,
			Solved:     stat.Solved,
			SolvedBy:   stat.SolvedBy,
			Penalty:    stat.Penalty,
			SolvedAt:   stat.SolvedAt,
		})
	}
	playerInfos := make([]types.TeamRoomPlayerInfo, 0, len(players))
	for _, p := range players {
		playerInfos = append(playerInfos, types.TeamRoomPlayerInfo{
			UserID:   p.UserID,
			Username: p.Username,
			JoinAt:   p.JoinAt,
		})
	}
	submissionInfos := make([]types.TeamRoomSubmissionInfo, 0, len(submissions))
	for _, s := range submissions {
		submissionInfos = append(submissionInfos, types.TeamRoomSubmissionInfo{
			SubmissionID: s.SubmissionID,
			ProblemID:    s.ProblemID,
			UserID:       s.UserID,
			Verdict:      s.Verdict,
			SubmitTime:   s.SubmitTime,
		})
	}
	return types.TeamRoomInfo{
		RoomID:      room.ID,
		Mode:        room.Mode,
		Status:      room.Status,
		CreatedAt:   room.CreatedAt,
		EndTime:     room.EndTime,
		Players:     playerInfos,
		Problems:    problemInfos,
		Submissions: submissionInfos,
		Score:       extra.Score,
		Duration:    extra.DurationSeconds,
	}
}

func parseTeamRoomProblems(value string) []teamRoomProblem {
	if value == "" {
		return nil
	}
	var items []teamRoomProblem
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseTeamRoomProblemStatus(value string) []teamRoomProblemStatus {
	if value == "" {
		return nil
	}
	var items []teamRoomProblemStatus
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseTeamRoomPlayers(value string) []teamRoomPlayer {
	if value == "" {
		return nil
	}
	var items []teamRoomPlayer
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseTeamRoomSubmissions(value string) []teamRoomSubmissionRecord {
	if value == "" {
		return nil
	}
	var items []teamRoomSubmissionRecord
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseTeamRoomExtra(value string) teamRoomExtraInfo {
	if value == "" {
		return teamRoomExtraInfo{}
	}
	var extra teamRoomExtraInfo
	if err := json.Unmarshal([]byte(value), &extra); err != nil {
		return teamRoomExtraInfo{}
	}
	return extra
}

type teamRoomProblem struct {
	ProblemID  string `json:"problem_id"`
	ProblemURL string `json:"problem_url"`
	Difficulty int    `json:"difficulty"`
}

type teamRoomPlayer struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	JoinAt   int64  `json:"join_at"`
}

type teamRoomSubmissionRecord struct {
	SubmissionID int64  `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	UserID       int64  `json:"user_id"`
	Verdict      string `json:"verdict"`
	SubmitTime   int64  `json:"submit_time"`
}

type teamRoomProblemStatus struct {
	ProblemID string `json:"problem_id"`
	Solved    bool   `json:"solved"`
	SolvedBy  int64  `json:"solved_by"`
	Penalty   int    `json:"penalty"`
	SolvedAt  int64  `json:"solved_at"`
}

type teamRoomExtraInfo struct {
	Score           int64 `json:"score"`
	DurationSeconds int64 `json:"duration_seconds"`
}

const teamRoomDefaultDuration = 5 * time.Hour

type teamRoomModeConfig struct {
	Mode     string
	Duration time.Duration
	Problems []int
}

var teamRoomModeConfigs = map[string]teamRoomModeConfig{
	"div3": {
		Mode:     "div3",
		Duration: 3 * time.Hour,
		Problems: []int{800, 800, 900, 1000, 1200, 1400, 1600, 1800, 2000},
	},
	"div3-plus": {
		Mode:     "div3-plus",
		Duration: 3 * time.Minute,
		Problems: []int{800, 800, 900, 900, 1000, 1100, 1100, 1200, 1300, 1400, 1500, 1600, 1800, 1900, 2100},
	},
}

func getTeamRoomModeConfig(mode string) (teamRoomModeConfig, bool) {
	config, ok := teamRoomModeConfigs[mode]
	return config, ok
}

func getTeamRoomDuration(mode string) time.Duration {
	if config, ok := teamRoomModeConfigs[mode]; ok && config.Duration > 0 {
		return config.Duration
	}
	return teamRoomDefaultDuration
}

func buildTeamRoomModeInfos() []types.TeamRoomModeInfo {
	if len(teamRoomModeConfigs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(teamRoomModeConfigs))
	for key := range teamRoomModeConfigs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]types.TeamRoomModeInfo, 0, len(keys))
	for _, key := range keys {
		config := teamRoomModeConfigs[key]
		problems := make([]int, len(config.Problems))
		copy(problems, config.Problems)
		items = append(items, types.TeamRoomModeInfo{
			Mode:     config.Mode,
			Duration: int64(getTeamRoomDuration(config.Mode).Seconds()),
			Problems: problems,
		})
	}
	return items
}
