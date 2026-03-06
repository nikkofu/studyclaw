package models

import (
	"fmt"
	"log"

	"github.com/nikkofu/studyclaw/api-server/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.GetEnv("DB_USER"),
		config.GetEnv("DB_PASSWORD"),
		config.GetEnv("DB_HOST"),
		config.GetEnv("DB_PORT"),
		config.GetEnv("DB_NAME"),
	)

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to MySQL: %v. Running in Mock Mode without Database.", err)
		DB = nil
		return
	}

	// Auto-migrate tables
	err = database.AutoMigrate(&User{}, &Family{}, &Task{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	DB = database
	log.Println("Database connection successfully established and migrated.")
}
