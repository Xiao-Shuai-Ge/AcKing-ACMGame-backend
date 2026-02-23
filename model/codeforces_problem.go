package model

type CodeforcesProblem struct {
	ID         string `gorm:"column:id;type:varchar(32);primaryKey"`
	Url        string `gorm:"column:url;type:varchar(255);not null"`
	Difficulty int    `gorm:"column:difficulty;type:int;default:0"`
}

func (c *CodeforcesProblem) TableName() string {
	return "codeforces_problems"
}
