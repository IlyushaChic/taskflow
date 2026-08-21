package analytics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

type Event struct {
	TaskID    uuid.UUID
	EventType string
	Status    string
	Assignee  string
}

type ClickHouseAnalyticsClient struct {
	conn    clickhouse.Conn
	events  chan Event
	wg      sync.WaitGroup
	closeCh chan struct{}
}

// NewClickHouseAnalyticsClient создаёт клиент с явными параметрами
func NewClickHouseAnalyticsClient(host string) (*ClickHouseAnalyticsClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{host}, // "localhost:9000"
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "", // пустой пароль по умолчанию
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}
	c := &ClickHouseAnalyticsClient{
		conn:    conn,
		events:  make(chan Event, 1000),
		closeCh: make(chan struct{}),
	}
	c.wg.Add(1)
	go c.worker()
	return c, nil
}

func (c *ClickHouseAnalyticsClient) worker() {
	defer c.wg.Done()
	batchSize := 100
	buffer := make([]Event, 0, batchSize)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		ctx := context.Background()
		batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO task_analytics (task_id, event_type, status, assignee, timestamp)")
		if err != nil {
			log.Printf("Analytics batch prepare error: %v", err)
			return
		}
		now := time.Now().UTC()
		for _, ev := range buffer {
			if err := batch.Append(ev.TaskID, ev.EventType, ev.Status, ev.Assignee, now); err != nil {
				log.Printf("Analytics batch append error: %v", err)
			}
		}
		if err := batch.Send(); err != nil {
			log.Printf("Analytics batch send error: %v", err)
		} else {
			log.Printf("Analytics: flushed %d events", len(buffer))
		}
		buffer = buffer[:0]
	}

	for {
		select {
		case ev := <-c.events:
			buffer = append(buffer, ev)
			if len(buffer) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-c.closeCh:
			for len(c.events) > 0 {
				ev := <-c.events
				buffer = append(buffer, ev)
			}
			flush()
			return
		}
	}
}

func (c *ClickHouseAnalyticsClient) SendEvent(ev Event) {
	select {
	case c.events <- ev:
	default:
		log.Println("Analytics event dropped (buffer full)")
	}
}

func (c *ClickHouseAnalyticsClient) Close() {
	close(c.closeCh)
	c.wg.Wait()
	c.conn.Close()
}

func (c *ClickHouseAnalyticsClient) GetStats(ctx context.Context) (map[string]int64, error) {
	query := `
        SELECT status, COUNT(*) as cnt
        FROM task_analytics
        WHERE timestamp >= now() - INTERVAL 1 DAY
        GROUP BY status
    `
	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var cnt uint64
		if err := rows.Scan(&status, &cnt); err != nil {
			return nil, err
		}
		stats[status] = int64(cnt)
	}
	return stats, nil
}
