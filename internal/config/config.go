package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort       string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	UploadDir        string
	OnlyOfficeAPIURL string // URL OnlyOffice Document Server (например, http://onlyoffice:8080)
	BaseURL          string // URL нашего сервиса, доступный для OnlyOffice (например, http://localhost:8082)
}

func Load() (*Config, error) {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8082"
	}

	return &Config{
		ServerPort:       port,
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "docservice"),
		DBPassword:       getEnv("DB_PASSWORD", "docservice"),
		DBName:           getEnv("DB_NAME", "docservice_db"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		UploadDir:        getEnv("UPLOAD_DIR", "./uploads"),
		OnlyOfficeAPIURL: getEnv("ONLYOFFICE_API_URL", "http://onlyoffice:8080"),
		BaseURL:          getEnv("BASE_URL", "http://localhost:8082"),
	}, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
