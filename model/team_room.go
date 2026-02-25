package model

type TeamRoom struct {
	CommonModel
	Mode              string `gorm:"column:mode;type:varchar(32);not null;index:idx_team_room_mode;comment:模式"`
	ProblemList       string `gorm:"column:problem_list;type:json;comment:题目列表JSON"`
	PlayerList        string `gorm:"column:player_list;type:json;comment:参与玩家列表JSON"`
	CreatorID         int64  `gorm:"column:creator_id;type:bigint;not null;index:idx_team_room_creator_id;comment:创建人ID"`
	EndTime           int64  `gorm:"column:end_time;type:bigint;default:0;index:idx_team_room_end_time;comment:结束时间戳"`
	Status            int8   `gorm:"column:status;type:tinyint;default:0;index:idx_team_room_status;comment:房间状态(0进行中,1结束)"`
	SubmissionRecords string `gorm:"column:submission_records;type:json;comment:提交记录JSON"`
	ProblemStatus     string `gorm:"column:problem_status;type:json;comment:题目情况JSON"`
	ExtraInfo         string `gorm:"column:extra_info;type:json;comment:额外信息JSON"`
}

func (t *TeamRoom) TableName() string {
	return "team_room"
}
