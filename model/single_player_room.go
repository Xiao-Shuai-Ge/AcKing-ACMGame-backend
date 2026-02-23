package model

type SinglePlayerRoom struct {
	CommonModel
	ProblemID        string `gorm:"column:problem_id;type:varchar(32);not null;index:idx_single_player_room_problem_id;comment:题目ID"`
	UserID           int64  `gorm:"column:user_id;type:bigint;not null;index:idx_single_player_room_user_id;comment:玩家ID"`
	EndTime          int64  `gorm:"column:end_time;type:bigint;default:0;index:idx_single_player_room_end_time;comment:结束时间戳"`
	Status           int8   `gorm:"column:status;type:tinyint;default:0;index:idx_single_player_room_status;comment:完成状态(0进行中,1放弃,2AC)"`
	Penalty          int    `gorm:"column:penalty;type:int;default:0;comment:罚时"`
	PerformanceScore int    `gorm:"column:performance_score;type:int;default:0;comment:表现分"`
	RatingBefore     int    `gorm:"column:rating_before;type:int;default:0;comment:结算前rating"`
	RatingAfter      int    `gorm:"column:rating_after;type:int;default:0;comment:结算后rating"`
}

func (s *SinglePlayerRoom) TableName() string {
	return "single_player_room"
}
