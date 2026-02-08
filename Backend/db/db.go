package db

import (
    "github.com/glebarez/sqlite" 
    "gorm.io/gorm"
    "log"
)

var DB *gorm.DB

func InitDB() {
	var err error
	// connect SQLite 
	DB, err = gorm.Open(sqlite.Open("homework.db"), &gorm.Config{})//自己的db文件路径
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB.Exec("PRAGMA foreign_keys = ON;")
	
	log.Println("Database connected and foreign keys enabled.")
}