package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	AppPort        string
	AppName        string
	DatabaseURL    string
	DBMaxConns     int32
	DBMinConns     int32
	DBMaxConnIdle  time.Duration
	DBMaxConnLife  time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load() // ignore error if .env not present (e.g. in docker or production)

	appEnv := getEnv("APP_ENV", "development")
	appPort := getEnv("APP_PORT", "8080")
	appName := getEnv("APP_NAME", "mergiate-core")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5434")
	dbUser := getEnv("DB_USER", "mergiate")
	dbPass := getEnv("DB_PASSWORD", "mergiate_password")
	dbName := getEnv("DB_NAME", "mergiate_core")
	dbSSL := getEnv("DB_SSLMODE", "disable")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPass, dbHost, dbPort, dbName, dbSSL,
		)
	}

	maxConns := getEnvAsInt32("DB_MAX_CONNS", 25)
	minConns := getEnvAsInt32("DB_MIN_CONNS", 5)

	return &Config{
		AppEnv:        appEnv,
		AppPort:       appPort,
		AppName:       appName,
		DatabaseURL:   dbURL,
		DBMaxConns:    maxConns,
		DBMinConns:    minConns,
		DBMaxConnIdle: 15 * time.Minute,
		DBMaxConnLife: 1 * time.Hour,
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt32(key string, fallback int32) int32 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(val)
}
