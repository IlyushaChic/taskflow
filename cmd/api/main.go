package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"taskflow/internal/analytics"
	"taskflow/internal/cache"
	"taskflow/internal/config"
	"taskflow/internal/handlers"
	"taskflow/internal/hub"
	"taskflow/internal/kafka"
	"taskflow/internal/repository"
	"taskflow/internal/services"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Загрузка .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
	cfg := config.Load()

	// Sentry
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.SentryDSN,
		}); err != nil {
			log.Printf("Sentry initialization failed: %v", err)
		} else {
			log.Println("Sentry initialized")
		}
		defer sentry.Flush(2 * time.Second)
	}

	// PostgreSQL
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("PostgreSQL connected")

	// Redis
	var redisCache cache.Cache
	if cfg.RedisURL != "" {
		rc, err := cache.NewRedisCache(cfg.RedisURL)
		if err != nil {
			log.Printf("Redis initialization failed: %v, caching disabled", err)
		} else {
			redisCache = rc
			log.Println("Redis connected, caching enabled")
		}
	} else {
		log.Println("REDIS_URL not set, caching disabled")
	}

	// Outbox репозиторий
	outboxRepo := repository.NewOutboxRepository(pool)

	// Базовый репозиторий с outbox
	baseRepo := repository.NewTaskRepository(pool, outboxRepo)

	// Декоратор кеша
	var taskRepo repository.TaskRepository
	if redisCache != nil {
		taskRepo = repository.NewTaskRepositoryCache(baseRepo, redisCache)
		log.Println("TaskRepository with cache decorator created")
	} else {
		taskRepo = baseRepo
		log.Println("TaskRepository without cache (fallback)")
	}

	// ClickHouse
	var analyticsClient *analytics.ClickHouseAnalyticsClient
	if cfg.ClickHouseDSN != "" {
		ac, err := analytics.NewClickHouseAnalyticsClient(cfg.ClickHouseDSN)
		if err != nil {
			log.Printf("ClickHouse initialization failed: %v, analytics disabled", err)
		} else {
			analyticsClient = ac
			log.Println("ClickHouse connected, analytics enabled")
		}
	} else {
		log.Println("CLICKHOUSE_DSN not set, analytics disabled")
	}

	// Kafka producer
	var kafkaProducer *kafka.Producer
	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		producer, err := kafka.NewProducer(brokers, "task-events")
		if err != nil {
			log.Printf("Kafka producer initialization failed: %v", err)
		} else {
			kafkaProducer = producer
			log.Println("Kafka producer connected")
		}
	} else {
		log.Println("KAFKA_BROKERS not set, Kafka disabled")
	}

	// Сервис и хендлеры
	taskService := services.NewTaskService(taskRepo)

	// WebSocket Hub
	wsHub := hub.NewHub()
	go wsHub.Run()
	log.Println("WebSocket Hub started")

	taskHandler := handlers.NewTaskHandler(taskService, wsHub, analyticsClient, kafkaProducer)

	// Gin
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Sentry middleware
	r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	// WebSocket
	r.GET("/ws", func(c *gin.Context) {
		wsHub.ServeWS(c.Writer, c.Request)
	})

	// API routes
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

	// Server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}
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
