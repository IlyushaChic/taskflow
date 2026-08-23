package worker

import (
	"context"
	"log"
	"time"

	"taskflow/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type OutboxProcessor struct {
	outboxRepo   repository.OutboxRepository
	rabbitConn   *amqp.Connection
	rabbitChan   *amqp.Channel
	exchangeName string
	routingKey   string
	queueName    string
	batchSize    int
	interval     time.Duration
}

func NewOutboxProcessor(
	pool *pgxpool.Pool,
	rabbitURL string,
	exchange, queue, routingKey string,
	batchSize int,
	interval time.Duration,
) (*OutboxProcessor, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// Объявляем обменник и очередь
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return nil, err
	}
	q, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		return nil, err
	}

	return &OutboxProcessor{
		outboxRepo:   repository.NewOutboxRepository(pool),
		rabbitConn:   conn,
		rabbitChan:   ch,
		exchangeName: exchange,
		routingKey:   routingKey,
		queueName:    q.Name,
		batchSize:    batchSize,
		interval:     interval,
	}, nil
}

// Run запускает основной цикл обработки
func (p *OutboxProcessor) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.processBatch(ctx)
		case <-ctx.Done():
			log.Println("Stopping outbox processor...")
			return
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context) {
	msgs, err := p.outboxRepo.GetUnprocessed(ctx, p.batchSize)
	if err != nil {
		log.Printf("GetUnprocessed error: %v", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	for _, out := range msgs {
		err := p.rabbitChan.PublishWithContext(
			ctx,
			p.exchangeName,
			p.routingKey,
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
		if err := p.outboxRepo.MarkProcessed(ctx, out.ID); err != nil {
			log.Printf("MarkProcessed error for outbox %d: %v", out.ID, err)
		} else {
			log.Printf("Processed outbox %d, event %s", out.ID, out.EventType)
		}
	}
}

func (p *OutboxProcessor) Close() error {
	if err := p.rabbitChan.Close(); err != nil {
		return err
	}
	return p.rabbitConn.Close()
}
