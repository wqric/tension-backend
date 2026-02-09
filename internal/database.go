package internal

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDatabase() error {
	var err error
	db, err = gorm.Open(sqlite.Open("users.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(&User{}, &Workout{}, &Exercises{}, &UserWorkout{})
}

func GetDB() *gorm.DB {
	return db
}
