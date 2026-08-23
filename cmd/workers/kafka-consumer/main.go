package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"taskflow/internal/analytics"
	"taskflow/internal/config"
	"taskflow/internal/worker"
)

func main() {
	cfg := config.Load()

	// Подключение к ClickHouse
	analyticsClient, err := analytics.NewClickHouseAnalyticsClient(cfg.ClickHouseDSN)
	if err != nil {
		log.Fatalf("ClickHouse connection error: %v", err)
	}
	defer analyticsClient.Close()
	log.Println("ClickHouse connected")

	// Создаём Kafka consumer
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	consumer, err := worker.NewKafkaConsumer(brokers, cfg.KafkaGroup, cfg.KafkaTopic, analyticsClient)
	if err != nil {
		log.Fatalf("Kafka consumer init error: %v", err)
	}
	defer consumer.Close()
	log.Println("Kafka consumer started, listening for events...")

	// Контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем потребителя в горутине
	go consumer.Run(ctx)

	// Ожидание сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down Kafka consumer gracefully...")
	cancel()

	// Даём время завершить текущие операции
	time.Sleep(2 * time.Second)
	log.Println("Kafka consumer stopped")
}
