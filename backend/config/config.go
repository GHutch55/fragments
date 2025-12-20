package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabasePath string
	JWTSecret    string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		return nil, errors.New("DATABASE_PATH environment variable is required")
	}

	JWTsecret := os.Getenv("JWT_SECRET")
	if JWTsecret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:         port,
		DatabasePath: dbPath,
		JWTSecret:    JWTsecret,
	}, nil
}
