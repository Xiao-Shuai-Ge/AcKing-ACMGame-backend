package repo

import (
	"gorm.io/gorm"
	"tgwp/model"
)

type UserRepo struct {
	DB *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{DB: db}
}

func (r *UserRepo) GetByEmail(email string) (model.User, error) {
	var user model.User
	err := r.DB.Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *UserRepo) GetByID(id int64) (model.User, error) {
	var user model.User
	err := r.DB.Where("id = ?", id).First(&user).Error
	return user, err
}

func (r *UserRepo) Create(user *model.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepo) UpdateProfile(user model.User) error {
	return r.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"username": user.Username,
	}).Error
}

func (r *UserRepo) UpdateRating(id int64, rating int) error {
	return r.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"rating": rating,
	}).Error
}
