package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBURL               string
	RedisURL            string
	ServerPort          string
	ClickHouseDSN       string
	KafkaBrokers        string
	KafkaTopic          string
	KafkaGroup          string
	JaegerEndpoint      string
	RabbitMQURL         string
	OutboxFetchInterval time.Duration
	OutboxBatchSize     int
}

func Load() Config {
	return Config{
		DBURL:               getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/taskflow?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", ""),
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		ClickHouseDSN:       getEnv("CLICKHOUSE_DSN", ""),
		KafkaBrokers:        getEnv("KAFKA_BROKERS", ""),
		KafkaTopic:          getEnv("KAFKA_TOPIC", "task-events"),
		KafkaGroup:          getEnv("KAFKA_GROUP", "taskflow-analytics-group"),
		JaegerEndpoint:      getEnv("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		OutboxFetchInterval: getEnvDuration("OUTBOX_FETCH_INTERVAL", 5*time.Second),
		OutboxBatchSize:     getEnvInt("OUTBOX_BATCH_SIZE", 10),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
