package repository

import (
	"context"
	"taskflow/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) OutboxRepository {
	return &outboxRepo{pool: pool}
}

func (r *outboxRepo) SaveInTransaction(ctx context.Context, tx pgx.Tx, outbox *models.Outbox) error {
	query := `INSERT INTO outbox (aggregate_id, event_type, payload) VALUES ($1, $2, $3)`
	_, err := tx.Exec(ctx, query, outbox.AggregateID, outbox.EventType, outbox.Payload)
	return err
}

func (r *outboxRepo) GetUnprocessed(ctx context.Context, limit int) ([]models.Outbox, error) {
	query := `SELECT id, aggregate_id, event_type, payload, created_at, processed_at
              FROM outbox
              WHERE processed_at IS NULL
              ORDER BY created_at
              LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var outboxes []models.Outbox
	for rows.Next() {
		var o models.Outbox
		err := rows.Scan(&o.ID, &o.AggregateID, &o.EventType, &o.Payload, &o.CreatedAt, &o.ProcessedAt)
		if err != nil {
			return nil, err
		}
		outboxes = append(outboxes, o)
	}
	return outboxes, nil
}

func (r *outboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET processed_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
