package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"taskflow/internal/cache"
	"taskflow/internal/config"
	"taskflow/internal/handlers"
	"taskflow/internal/repository"
	"taskflow/internal/services"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
	cfg := config.Load()

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

	// Подключение к PostgreSQL
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("PostgreSQL connected")

	// Подключение к Redis
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

	// Создаём outbox репозиторий
	outboxRepo := repository.NewOutboxRepository(pool)

	// Создаём базовый репозиторий с outbox
	baseRepo := repository.NewTaskRepository(pool, outboxRepo)

	// Оборачиваем в кеш, если Redis доступен
	var taskRepo repository.TaskRepository
	if redisCache != nil {
		taskRepo = repository.NewTaskRepositoryCache(baseRepo, redisCache)
		log.Println("TaskRepository with cache decorator created")
	} else {
		taskRepo = baseRepo
		log.Println("TaskRepository without cache (fallback)")
	}

	// Сервис и хендлеры
	taskService := services.NewTaskService(taskRepo)
	taskHandler := handlers.NewTaskHandler(taskService)

	// Настройка роутера Gin
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Sentry middleware (ловит паники и отправляет в Sentry)
	r.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	// Роуты API
	api := r.Group("/api/v1")
	{
		api.GET("/tasks", taskHandler.List)
		api.POST("/tasks", taskHandler.Create)
		api.GET("/tasks/:id", taskHandler.Get)
		api.PUT("/tasks/:id", taskHandler.Update)
		api.DELETE("/tasks/:id", taskHandler.Delete)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Запуск сервера
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
