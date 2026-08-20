package repository

import (
	"context"
	"taskflow/internal/models"

	"github.com/jackc/pgx/v5"
)

type OutboxRepository interface {
	// SaveInTransaction сохраняет запись outbox в рамках переданной транзакции
	SaveInTransaction(ctx context.Context, tx pgx.Tx, outbox *models.Outbox) error
	// GetUnprocessed возвращает необработанные сообщения (можно лимитировать)
	GetUnprocessed(ctx context.Context, limit int) ([]models.Outbox, error)
	// MarkProcessed помечает сообщение как обработанное
	MarkProcessed(ctx context.Context, id int64) error
}
