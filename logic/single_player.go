package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"tgwp/global"
	"tgwp/model"
	"tgwp/repo"
	"tgwp/response"
	"tgwp/types"
)

type SinglePlayerLogic struct {
}

func NewSinglePlayerLogic() *SinglePlayerLogic {
	return &SinglePlayerLogic{}
}

func (l *SinglePlayerLogic) CreateRoom(ctx context.Context, userID int64) (resp types.SinglePlayerCreateResp, err error) {
	_ = ctx
	if userID == 0 {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	userRepo := repo.NewUserRepo(global.DB)
	user, err := userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MEMBER_NOT_EXIST)
		}
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	roomRepo := repo.NewSinglePlayerRoomRepo(global.DB)
	activeRoom, err := roomRepo.GetActiveByUser(userID)
	if err == nil && activeRoom.ID != 0 {
		problem, err := repo.NewCodeforcesProblemRepo(global.DB).GetByID(activeRoom.ProblemID)
		if err != nil {
			return resp, response.ErrResp(err, response.DATABASE_ERROR)
		}
		GetSinglePlayerManager().StartRoom(activeRoom, problem)
		resp.Room = buildSingleRoomInfo(activeRoom, problem)
		return resp, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	rating := user.Rating
	if rating <= 0 {
		rating = 800
	}
	minDifficulty := rating - 200
	if minDifficulty < 0 {
		minDifficulty = 0
	}
	maxDifficulty := rating + 200
	problem, err := repo.NewCodeforcesProblemRepo(global.DB).GetRandomByDifficulty(minDifficulty, maxDifficulty)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	// 调试，指定同一道题目
	// problem.ID = "1541A"
	room := model.SinglePlayerRoom{
		ProblemID:    problem.ID,
		UserID:       userID,
		RatingBefore: rating,
	}
	if err := roomRepo.Create(&room); err != nil {
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	GetSinglePlayerManager().StartRoom(room, problem)
	resp.Room = buildSingleRoomInfo(room, problem)
	return resp, nil
}

func (l *SinglePlayerLogic) GetRoomInfo(ctx context.Context, req types.SinglePlayerRoomInfoReq) (resp types.SinglePlayerRoomInfoResp, err error) {
	_ = ctx
	roomID, err := parseRoomID(req.RoomID)
	if err != nil {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	room, problem, err := l.getRoomAndProblem(roomID)
	if err != nil {
		return resp, err
	}
	resp.Room = buildSingleRoomInfo(room, problem)
	return resp, nil
}

func (l *SinglePlayerLogic) AbandonRoom(ctx context.Context, userID int64, req types.SinglePlayerAbandonReq) (resp types.SinglePlayerAbandonResp, err error) {
	if userID == 0 {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	roomRepo := repo.NewSinglePlayerRoomRepo(global.DB)
	var room model.SinglePlayerRoom
	if req.RoomID != "" {
		roomID, err := parseRoomID(req.RoomID)
		if err != nil {
			return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
		}
		room, err = roomRepo.GetByID(roomID)
	} else {
		room, err = roomRepo.GetActiveByUser(userID)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	if room.UserID != userID {
		return resp, response.ErrResp(errors.New("permission denied"), response.PERMISSION_DENIED)
	}
	problem, err := repo.NewCodeforcesProblemRepo(global.DB).GetByID(room.ProblemID)
	if err != nil {
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	if room.Status == 0 {
		room, err = finishSingleRoom(ctx, room, problem.Difficulty, room.Penalty, 1)
		if err != nil {
			return resp, err
		}
		GetSinglePlayerManager().StopRoom(room.ID)
		GetWsHub().SendToUser(room.UserID, types.WsResponse{
			Type:    "single_room_finish",
			Code:    response.SUCCESS.Code,
			Message: response.SUCCESS.Msg,
			Data: map[string]interface{}{
				"room": buildSingleRoomInfo(room, problem),
			},
		})
	}
	resp.Room = buildSingleRoomInfo(room, problem)
	return resp, nil
}

func (l *SinglePlayerLogic) getRoomAndProblem(roomID int64) (model.SinglePlayerRoom, model.CodeforcesProblem, error) {
	roomRepo := repo.NewSinglePlayerRoomRepo(global.DB)
	room, err := roomRepo.GetByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return room, model.CodeforcesProblem{}, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return room, model.CodeforcesProblem{}, response.ErrResp(err, response.DATABASE_ERROR)
	}
	problem, err := repo.NewCodeforcesProblemRepo(global.DB).GetByID(room.ProblemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return room, model.CodeforcesProblem{}, response.ErrResp(err, response.MESSAGE_NOT_EXIST)
		}
		return room, model.CodeforcesProblem{}, response.ErrResp(err, response.DATABASE_ERROR)
	}
	return room, problem, nil
}

func parseRoomID(roomID string) (int64, error) {
	if roomID == "" {
		return 0, errors.New("param blank")
	}
	return strconv.ParseInt(roomID, 10, 64)
}

func buildSingleRoomInfo(room model.SinglePlayerRoom, problem model.CodeforcesProblem) types.SinglePlayerRoomInfo {
	info := types.SinglePlayerRoomInfo{
		RoomID:            room.ID,
		UserID:            room.UserID,
		ProblemID:         room.ProblemID,
		ProblemURL:        problem.Url,
		ProblemDifficulty: problem.Difficulty,
		Status:            room.Status,
		Penalty:           room.Penalty,
		PerformanceScore:  room.PerformanceScore,
		RatingBefore:      room.RatingBefore,
		RatingAfter:       room.RatingAfter,
		CreatedAt:         room.CreatedAt,
		EndTime:           room.EndTime,
	}

	if room.ExtraInfo != "" {
		var extra struct {
			Submissions []types.RoomSubmissionRecord `json:"submissions"`
		}
		if err := json.Unmarshal([]byte(room.ExtraInfo), &extra); err == nil {
			info.Submissions = extra.Submissions
		}
	}
	return info
}

func calcPerformance(difficulty int, minutes int, penalty int, solved bool) int {
	if !solved {
		return difficulty - 200
	}
	totalMinutes := minutes + penalty
	over := totalMinutes - 10
	if over < 0 {
		over = 0
	}
	score := difficulty + 200 - over*10
	minScore := difficulty - 100
	if minScore < score {
		score = minScore
	}
	return score
}

func finishSingleRoom(ctx context.Context, room model.SinglePlayerRoom, difficulty int, penalty int, status int8) (model.SinglePlayerRoom, error) {
	_ = ctx
	solved := status == 2
	minutes := int(time.Since(room.CreatedAt).Minutes())
	performance := calcPerformance(difficulty, minutes, penalty, solved)
	ratingBefore := room.RatingBefore
	if ratingBefore == 0 {
		user, err := repo.NewUserRepo(global.DB).GetByID(room.UserID)
		if err == nil {
			ratingBefore = user.Rating
		}
	}
	if ratingBefore == 0 {
		ratingBefore = 800
	}
	ratingAfter := (performance + ratingBefore) / 2
	endTime := time.Now().Unix()
	roomRepo := repo.NewSinglePlayerRoomRepo(global.DB)
	if err := roomRepo.FinishRoom(room.ID, status, endTime, performance, ratingBefore, ratingAfter, penalty); err != nil {
		return room, response.ErrResp(err, response.DATABASE_ERROR)
	}
	if err := repo.NewUserRepo(global.DB).UpdateRating(room.UserID, ratingAfter); err != nil {
		return room, response.ErrResp(err, response.DATABASE_ERROR)
	}
	room.Status = status
	room.EndTime = endTime
	room.PerformanceScore = performance
	room.RatingBefore = ratingBefore
	room.RatingAfter = ratingAfter
	room.Penalty = penalty
	return room, nil
}
