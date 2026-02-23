package model

type User struct {
	CommonModel
	Email    string `gorm:"column:email;type:varchar(255);not null;uniqueIndex"`
	Password string `gorm:"column:password;type:varchar(255);not null"`
	Username string `gorm:"column:username;type:varchar(100);not null"`
	Rating   int    `gorm:"column:rating;type:int;default:800"`
}

func (u *User) TableName() string {
	return "users"
}
