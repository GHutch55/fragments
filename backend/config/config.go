package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL environment variable is required")
	}

	JWTsecret := os.Getenv("JWT_SECRET")
	if JWTsecret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:         port,
		DatabaseURL: dbURL,
		JWTSecret:    JWTsecret,
	}, nil
}
