package repo

import (
	"tgwp/model"

	"gorm.io/gorm"
)

type TeamRoomRepo struct {
	DB *gorm.DB
}

func NewTeamRoomRepo(db *gorm.DB) *TeamRoomRepo {
	return &TeamRoomRepo{DB: db}
}

func (r *TeamRoomRepo) Create(room *model.TeamRoom) error {
	return r.DB.Create(room).Error
}

func (r *TeamRoomRepo) GetByID(id int64) (model.TeamRoom, error) {
	var room model.TeamRoom
	err := r.DB.Where("id = ?", id).First(&room).Error
	return room, err
}

func (r *TeamRoomRepo) List(offset, limit int, status *int8) ([]model.TeamRoom, int64, error) {
	query := r.DB.Model(&model.TeamRoom{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	var rooms []model.TeamRoom
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&rooms).Error
	return rooms, count, err
}

func (r *TeamRoomRepo) ListActive() ([]model.TeamRoom, error) {
	var rooms []model.TeamRoom
	err := r.DB.Where("status = ?", 0).Order("created_at asc").Find(&rooms).Error
	return rooms, err
}

func (r *TeamRoomRepo) UpdateStatus(id int64, status int8, endTime int64) error {
	return r.DB.Model(&model.TeamRoom{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   status,
		"end_time": endTime,
	}).Error
}

func (r *TeamRoomRepo) UpdateSubmissionRecords(id int64, value string) error {
	return r.DB.Model(&model.TeamRoom{}).Where("id = ?", id).Update("submission_records", value).Error
}

func (r *TeamRoomRepo) UpdateProblemStatus(id int64, value string) error {
	return r.DB.Model(&model.TeamRoom{}).Where("id = ?", id).Update("problem_status", value).Error
}

func (r *TeamRoomRepo) UpdatePlayerList(id int64, value string) error {
	return r.DB.Model(&model.TeamRoom{}).Where("id = ?", id).Update("player_list", value).Error
}

func (r *TeamRoomRepo) UpdateExtraInfo(id int64, value string) error {
	return r.DB.Model(&model.TeamRoom{}).Where("id = ?", id).Update("extra_info", value).Error
}
