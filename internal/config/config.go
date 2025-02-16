package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	ServerPort string
	JWTSecret  string
}

func NewConfig() (*Config, error) {
	return &Config{
		DBHost:     getEnvOrDefault("DATABASE_HOST", "localhost"),
		DBPort:     getEnvOrDefault("DATABASE_PORT", "5432"),
		DBUser:     getEnvOrDefault("DATABASE_USER", "postgres"),
		DBPassword: getEnvOrDefault("DATABASE_PASSWORD", "password"),
		DBName:     getEnvOrDefault("DATABASE_NAME", "shop"),

		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		JWTSecret:  getEnvOrDefault("JWT_SECRET", "mysecret"),
	}, nil
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}
