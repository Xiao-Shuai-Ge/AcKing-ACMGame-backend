package repo

import (
	"tgwp/model"

	"gorm.io/gorm"
)

type SinglePlayerRoomRepo struct {
	DB *gorm.DB
}

func NewSinglePlayerRoomRepo(db *gorm.DB) *SinglePlayerRoomRepo {
	return &SinglePlayerRoomRepo{DB: db}
}

func (r *SinglePlayerRoomRepo) Create(room *model.SinglePlayerRoom) error {
	return r.DB.Create(room).Error
}

func (r *SinglePlayerRoomRepo) GetByID(id int64) (model.SinglePlayerRoom, error) {
	var room model.SinglePlayerRoom
	err := r.DB.Where("id = ?", id).First(&room).Error
	return room, err
}

func (r *SinglePlayerRoomRepo) GetActiveByUser(userID int64) (model.SinglePlayerRoom, error) {
	var room model.SinglePlayerRoom
	err := r.DB.Where("user_id = ? AND status = ?", userID, 0).Order("created_at desc").First(&room).Error
	return room, err
}

func (r *SinglePlayerRoomRepo) UpdatePenalty(id int64, penalty int) error {
	return r.DB.Model(&model.SinglePlayerRoom{}).Where("id = ?", id).Updates(map[string]interface{}{
		"penalty": penalty,
	}).Error
}

func (r *SinglePlayerRoomRepo) UpdateExtraInfo(id int64, extraInfo string) error {
	return r.DB.Model(&model.SinglePlayerRoom{}).Where("id = ?", id).Updates(map[string]interface{}{
		"extra_info": extraInfo,
	}).Error
}

func (r *SinglePlayerRoomRepo) FinishRoom(id int64, status int8, endTime int64, performance int, ratingBefore int, ratingAfter int, penalty int) error {
	return r.DB.Model(&model.SinglePlayerRoom{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":            status,
		"end_time":          endTime,
		"performance_score": performance,
		"rating_before":     ratingBefore,
		"rating_after":      ratingAfter,
		"penalty":           penalty,
	}).Error
}
