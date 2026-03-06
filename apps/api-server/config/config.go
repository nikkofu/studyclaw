package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

func LoadConfig() {
	viper.SetConfigFile("../../.env") // Load from root dir first
	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	} else {
		log.Println("Loaded environment from .env file")
	}

	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "studyclaw_dev_secret")
	viper.SetDefault("DB_NAME", "studyclaw_dev")
	viper.SetDefault("API_PORT", "8080")
}

func GetEnv(key string) string {
	// Let OS env override viper config
	if val := os.Getenv(key); val != "" {
		return val
	}
	return viper.GetString(key)
}
