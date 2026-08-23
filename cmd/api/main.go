package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"taskflow/internal/analytics"
	"taskflow/internal/cache"
	"taskflow/internal/config"
	"taskflow/internal/handlers"
	"taskflow/internal/hub"
	"taskflow/internal/kafka"
	"taskflow/internal/repository"
	"taskflow/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
	cfg := config.Load()

	// Инициализация компонентов
	pool := initPostgres(cfg)
	defer pool.Close()

	redisCache := initRedis(cfg)
	if redisCache != nil {
		defer func() { _ = redisCache.(interface{ Close() error }).Close() }()
	}

	analyticsClient := initClickHouse(cfg)
	if analyticsClient != nil {
		defer analyticsClient.Close()
	}

	kafkaProducer := initKafkaProducer(cfg)
	if kafkaProducer != nil {
		defer kafkaProducer.Close()
	}

	// Репозитории
	outboxRepo := repository.NewOutboxRepository(pool)
	baseRepo := repository.NewTaskRepository(pool, outboxRepo)

	var taskRepo repository.TaskRepository
	if redisCache != nil {
		taskRepo = repository.NewTaskRepositoryCache(baseRepo, redisCache)
	} else {
		taskRepo = baseRepo
	}

	// Сервисы и хендлеры
	taskService := services.NewTaskService(taskRepo)
	wsHub := hub.NewHub()
	go wsHub.Run()
	taskHandler := handlers.NewTaskHandler(taskService, wsHub, analyticsClient, kafkaProducer)

	// HTTP сервер
	r := setupRouter(cfg, wsHub, taskHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Запуск сервера
	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exited")
}

// initPostgres подключается к PostgreSQL
func initPostgres(cfg config.Config) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	log.Println("PostgreSQL connected")
	return pool
}

// initRedis подключается к Redis (если URL задан)
func initRedis(cfg config.Config) cache.Cache {
	if cfg.RedisURL == "" {
		log.Println("REDIS_URL not set, caching disabled")
		return nil
	}
	rc, err := cache.NewRedisCache(cfg.RedisURL)
	if err != nil {
		log.Printf("Redis initialization failed: %v, caching disabled", err)
		return nil
	}
	log.Println("Redis connected, caching enabled")
	return rc
}

// initClickHouse подключается к ClickHouse (если DSN задан)
func initClickHouse(cfg config.Config) *analytics.ClickHouseAnalyticsClient {
	if cfg.ClickHouseDSN == "" {
		log.Println("CLICKHOUSE_DSN not set, analytics disabled")
		return nil
	}
	ac, err := analytics.NewClickHouseAnalyticsClient(cfg.ClickHouseDSN)
	if err != nil {
		log.Printf("ClickHouse initialization failed: %v", err)
		return nil
	}
	log.Println("ClickHouse connected")
	return ac
}

// initKafkaProducer создаёт продюсера Kafka (если адрес задан)
func initKafkaProducer(cfg config.Config) *kafka.Producer {
	if cfg.KafkaBrokers == "" {
		log.Println("KAFKA_BROKERS not set, Kafka producer disabled")
		return nil
	}
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	producer, err := kafka.NewProducer(brokers, "task-events")
	if err != nil {
		log.Printf("Kafka producer initialization failed: %v", err)
		return nil
	}
	log.Println("Kafka producer connected")
	return producer
}

// setupRouter настраивает Gin роутер и все маршруты
func setupRouter(cfg config.Config, wsHub *hub.Hub, taskHandler *handlers.TaskHandler) *gin.Engine {
	r := gin.Default()

	// CORS – разрешаем все источники (для разработки)
	r.Use(cors.Default())

	// WebSocket
	r.GET("/ws", func(c *gin.Context) {
		wsHub.ServeWS(c.Writer, c.Request)
	})

	// API
	api := r.Group("/api/v1")
	{
		api.GET("/tasks", taskHandler.List)
		api.POST("/tasks", taskHandler.Create)
		api.GET("/tasks/:id", taskHandler.Get)
		api.PUT("/tasks/:id", taskHandler.Update)
		api.DELETE("/tasks/:id", taskHandler.Delete)
		api.GET("/stats", taskHandler.GetStats)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}
