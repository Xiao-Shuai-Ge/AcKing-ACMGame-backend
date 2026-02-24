package types

import "time"

type SinglePlayerCreateReq struct {
}

type SinglePlayerCreateResp struct {
	Room SinglePlayerRoomInfo `json:"room"`
}

type SinglePlayerRoomInfoReq struct {
	RoomID string `form:"room_id" json:"room_id"`
}

type SinglePlayerRoomInfoResp struct {
	Room SinglePlayerRoomInfo `json:"room"`
}

type SinglePlayerAbandonReq struct {
	RoomID string `json:"room_id" form:"room_id"`
}

type SinglePlayerAbandonResp struct {
	Room SinglePlayerRoomInfo `json:"room"`
}

type SinglePlayerRoomInfo struct {
	RoomID          int64     `json:"room_id,string"`
	UserID          int64     `json:"user_id,string"`
	ProblemID       string    `json:"problem_id"`
	ProblemURL      string    `json:"problem_url"`
	ProblemDifficulty int     `json:"problem_difficulty"`
	Status          int8      `json:"status"`
	Penalty         int       `json:"penalty"`
	PerformanceScore int      `json:"performance_score"`
	RatingBefore    int       `json:"rating_before"`
	RatingAfter     int       `json:"rating_after"`
	CreatedAt       time.Time `json:"created_at"`
	EndTime         int64     `json:"end_time"`
}
