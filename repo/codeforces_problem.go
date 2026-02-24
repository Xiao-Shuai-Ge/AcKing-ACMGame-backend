package repo

import (
	"gorm.io/gorm"
	"tgwp/model"
)

type CodeforcesProblemRepo struct {
	DB *gorm.DB
}

func NewCodeforcesProblemRepo(db *gorm.DB) *CodeforcesProblemRepo {
	return &CodeforcesProblemRepo{DB: db}
}

func (r *CodeforcesProblemRepo) GetRandomByDifficulty(minDifficulty, maxDifficulty int) (model.CodeforcesProblem, error) {
	var problem model.CodeforcesProblem
	err := r.DB.Where("difficulty >= ? AND difficulty <= ?", minDifficulty, maxDifficulty).
		Order("RAND()").
		First(&problem).Error
	return problem, err
}

func (r *CodeforcesProblemRepo) GetByID(id string) (model.CodeforcesProblem, error) {
	var problem model.CodeforcesProblem
	err := r.DB.Where("id = ?", id).First(&problem).Error
	return problem, err
}
