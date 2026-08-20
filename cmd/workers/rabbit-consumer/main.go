package main

import (
	"context"
	"log"
	"time"

	"taskflow/internal/config"
	"taskflow/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg := config.Load()
	// Подключение к БД
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer pool.Close()

	outboxRepo := repository.NewOutboxRepository(pool)

	// Подключение к RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Channel error: %v", err)
	}
	defer ch.Close()

	// Объявляем обменник и очередь
	exchangeName := "task_events"
	queueName := "email_notifications"
	err = ch.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Exchange declare error: %v", err)
	}
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Queue declare error: %v", err)
	}
	err = ch.QueueBind(q.Name, "task.updated", exchangeName, false, nil)
	if err != nil {
		log.Fatalf("Queue bind error: %v", err)
	}

	// Цикл обработки outbox
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		msgs, err := outboxRepo.GetUnprocessed(context.Background(), 10)
		if err != nil {
			log.Printf("GetUnprocessed error: %v", err)
			continue
		}
		if len(msgs) == 0 {
			continue
		}
		for _, out := range msgs {
			// Отправляем в RabbitMQ
			err := ch.PublishWithContext(
				context.Background(),
				exchangeName,
				"task.updated",
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        out.Payload,
				},
			)
			if err != nil {
				log.Printf("Publish error for outbox %d: %v", out.ID, err)
				continue
			}
			// Помечаем как обработанное
			if err := outboxRepo.MarkProcessed(context.Background(), out.ID); err != nil {
				log.Printf("MarkProcessed error for outbox %d: %v", out.ID, err)
			} else {
				log.Printf("Processed outbox %d, event %s", out.ID, out.EventType)
			}
		}
	}
}
