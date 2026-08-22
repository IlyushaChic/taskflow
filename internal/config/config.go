package config

import (
	"os"
)

type Config struct {
	DBURL          string
	RedisURL       string
	ServerPort     string
	ClickHouseDSN  string
	KafkaBrokers   string
	JaegerEndpoint string
}

func Load() Config {
	return Config{
		DBURL:          getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/taskflow?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", ""),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		ClickHouseDSN:  getEnv("CLICKHOUSE_DSN", ""),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", ""),
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}
}
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
