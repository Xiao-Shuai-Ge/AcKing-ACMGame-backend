package model

import "gorm.io/gorm"

func MigrateTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&Template{},
		&User{},
		&CodeforcesProblem{},
	); err != nil {
		return err
	}
	return nil
}
