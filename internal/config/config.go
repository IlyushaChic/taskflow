package config

import (
	"os"
)

type Config struct {
	DBURL         string
	SentryDSN     string
	ServerPort    string
	RedisURL      string
	ClickHouseDSN string
}

func Load() Config {
	return Config{
		DBURL:         getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/taskflow?sslmode=disable"),
		SentryDSN:     getEnv("SENTRY_DSN", ""),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		RedisURL:      getEnv("REDIS_URL", ""),
		ClickHouseDSN: getEnv("CLICKHOUSE_DSN", "localhost:9000"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
