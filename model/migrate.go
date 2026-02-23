package model

import "gorm.io/gorm"

func MigrateTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&Template{},
		&User{},
	)
}
