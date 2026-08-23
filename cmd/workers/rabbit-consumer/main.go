package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskflow/internal/config"
	"taskflow/internal/worker"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	// Подключение к БД
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer pool.Close()

	// Создаём процессор outbox
	processor, err := worker.NewOutboxProcessor(
		pool,
		cfg.RabbitMQURL,
		"task_events",
		"email_notifications",
		"task.updated",
		cfg.OutboxBatchSize,
		cfg.OutboxFetchInterval,
	)
	if err != nil {
		log.Fatalf("Outbox processor init error: %v", err)
	}
	defer processor.Close()

	// Контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем обработку в горутине
	go processor.Run(ctx)

	// Ожидание сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
	cancel()

	// Даём время завершить текущие операции
	time.Sleep(2 * time.Second)
	log.Println("Worker stopped")
}
