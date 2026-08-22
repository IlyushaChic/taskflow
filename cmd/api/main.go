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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"taskflow/internal/analytics"
	"taskflow/internal/cache"
	"taskflow/internal/config"
	"taskflow/internal/handlers"
	"taskflow/internal/hub"
	"taskflow/internal/kafka"
	"taskflow/internal/observability"
	"taskflow/internal/repository"
	"taskflow/internal/services"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
	cfg := config.Load()

	// --- Observability ---
	// Tracer (Jaeger)
	var tracerProvider *sdktrace.TracerProvider
	if cfg.JaegerEndpoint != "" {
		tp, err := observability.InitTracer("taskflow-api", cfg.JaegerEndpoint)
		if err != nil {
			log.Printf("Jaeger initialization failed: %v", err)
		} else {
			tracerProvider = tp
			defer func() {
				if err := tracerProvider.Shutdown(context.Background()); err != nil {
					log.Printf("Tracer shutdown error: %v", err)
				}
			}()
			log.Println("Jaeger tracer initialized")
		}
	}

	// Metrics (Prometheus)
	if _, err := observability.InitMetrics(); err != nil {
		log.Printf("Prometheus metrics initialization failed: %v", err)
	} else {
		log.Println("Prometheus metrics exporter initialized")
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

	// Outbox, репозитории
	outboxRepo := repository.NewOutboxRepository(pool)
	baseRepo := repository.NewTaskRepository(pool, outboxRepo)
	var taskRepo repository.TaskRepository
	if redisCache != nil {
		taskRepo = repository.NewTaskRepositoryCache(baseRepo, redisCache)
	} else {
		taskRepo = baseRepo
	}

	// ClickHouse (опционально)
	var analyticsClient *analytics.ClickHouseAnalyticsClient
	if cfg.ClickHouseDSN != "" {
		ac, err := analytics.NewClickHouseAnalyticsClient(cfg.ClickHouseDSN)
		if err != nil {
			log.Printf("ClickHouse initialization failed: %v", err)
		} else {
			analyticsClient = ac
			log.Println("ClickHouse connected")
		}
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
	}

	// Сервисы
	taskService := services.NewTaskService(taskRepo)
	wsHub := hub.NewHub()
	go wsHub.Run()
	taskHandler := handlers.NewTaskHandler(taskService, wsHub, analyticsClient, kafkaProducer)

	// Gin роутер
	r := gin.Default()
	r.Use(cors.Default())

	// OpenTelemetry middleware (трассировка HTTP)
	r.Use(otelgin.Middleware("taskflow-api"))

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

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
