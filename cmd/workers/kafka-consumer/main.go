package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"taskflow/internal/analytics"
	"taskflow/internal/config"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type TaskEvent struct {
	TaskID    string `json:"task_id"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee"`
}

func main() {
	cfg := config.Load()

	// Подключение к ClickHouse
	analyticsClient, err := analytics.NewClickHouseAnalyticsClient(cfg.ClickHouseDSN)
	if err != nil {
		log.Fatalf("ClickHouse connection error: %v", err)
	}
	defer analyticsClient.Close()
	log.Println("ClickHouse connected")

	// Настройка Kafka consumer
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup([]string{"localhost:9092"}, "taskflow-analytics-group", config)
	if err != nil {
		log.Fatalf("Kafka consumer group error: %v", err)
	}
	defer consumerGroup.Close()

	log.Println("Kafka consumer started, listening for events...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
		log.Println("Shutting down Kafka consumer...")
		cancel()
	}()

	handler := &ConsumerHandler{
		analytics: analyticsClient,
	}

	for {
		if err := consumerGroup.Consume(ctx, []string{"task-events"}, handler); err != nil {
			log.Printf("Consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type ConsumerHandler struct {
	analytics *analytics.ClickHouseAnalyticsClient
}

func (h *ConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event TaskEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}
		// Парсим UUID
		taskID, err := uuid.Parse(event.TaskID)
		if err != nil {
			log.Printf("Invalid UUID: %v", err)
			continue
		}
		// Отправляем в ClickHouse
		h.analytics.SendEvent(analytics.Event{
			TaskID:    taskID,
			EventType: event.EventType,
			Status:    event.Status,
			Assignee:  event.Assignee,
		})
		session.MarkMessage(msg, "")
	}
	return nil
}
