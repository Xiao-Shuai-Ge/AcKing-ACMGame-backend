package model

import "gorm.io/gorm"

func MigrateTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&User{},
		&CodeforcesProblem{},
		&SinglePlayerRoom{},
		&TeamRoom{},
	); err != nil {
		return err
	}
	return nil
}
