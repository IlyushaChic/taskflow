package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"taskflow/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error)
	List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error)
	Update(ctx context.Context, task *models.Task, history []models.TaskHistory) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type taskRepo struct {
	pool       *pgxpool.Pool
	outboxRepo OutboxRepository
}

// NewTaskRepository теперь принимает OutboxRepository
func NewTaskRepository(pool *pgxpool.Pool, outboxRepo OutboxRepository) TaskRepository {
	return &taskRepo{
		pool:       pool,
		outboxRepo: outboxRepo,
	}
}

// Create – вставка задачи + outbox-событие в одной транзакции
func (r *taskRepo) Create(ctx context.Context, task *models.Task) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO tasks (id, title, description, status, assignee, due_date, version, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(ctx, query, task.ID, task.Title, task.Description, task.Status, task.Assignee, task.DueDate, task.Version, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return err
	}

	// Outbox-событие о создании
	payload, _ := json.Marshal(map[string]interface{}{
		"task_id": task.ID,
		"title":   task.Title,
		"status":  task.Status,
	})
	outbox := &models.Outbox{
		AggregateID: task.ID,
		EventType:   "task_created",
		Payload:     payload,
	}
	if err := r.outboxRepo.SaveInTransaction(ctx, tx, outbox); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetByID – без изменений (только чтение)
func (r *taskRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	query := `SELECT id, title, description, status, assignee, due_date, version, created_at, updated_at, deleted_at
              FROM tasks WHERE id = $1 AND deleted_at IS NULL`
	task := &models.Task{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status, &task.Assignee,
		&task.DueDate, &task.Version, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// List – без изменений (только чтение)
func (r *taskRepo) List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argPos := 1

	if filter.Status != "" {
		conditions = append(conditions, "status = $"+strconv.Itoa(argPos))
		args = append(args, filter.Status)
		argPos++
	}
	if filter.Assignee != "" {
		conditions = append(conditions, "assignee ILIKE '%' || $"+strconv.Itoa(argPos)+" || '%'")
		args = append(args, filter.Assignee)
		argPos++
	}
	if !filter.DueDateFrom.IsZero() {
		conditions = append(conditions, "due_date >= $"+strconv.Itoa(argPos))
		args = append(args, filter.DueDateFrom)
		argPos++
	}
	if !filter.DueDateTo.IsZero() {
		conditions = append(conditions, "due_date <= $"+strconv.Itoa(argPos))
		args = append(args, filter.DueDateTo)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, title, description, status, assignee, due_date, version, created_at, updated_at, deleted_at
              FROM tasks ` + whereClause + `
              ORDER BY created_at DESC
              LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var t models.Task
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Assignee,
			&t.DueDate, &t.Version, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, nil
}

// Update – обновление задачи + история + outbox в одной транзакции
func (r *taskRepo) Update(ctx context.Context, task *models.Task, history []models.TaskHistory) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	updateQuery := `UPDATE tasks
                    SET title = $1, description = $2, status = $3, assignee = $4, due_date = $5,
                        version = version + 1, updated_at = NOW()
                    WHERE id = $6 AND version = $7 AND deleted_at IS NULL`
	cmd, err := tx.Exec(ctx, updateQuery,
		task.Title, task.Description, task.Status, task.Assignee, task.DueDate,
		task.ID, task.Version)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if len(history) > 0 {
		histQuery := `INSERT INTO task_history (task_id, field_name, old_value, new_value) VALUES ($1, $2, $3, $4)`
		for _, h := range history {
			_, err := tx.Exec(ctx, histQuery, h.TaskID, h.FieldName, h.OldValue, h.NewValue)
			if err != nil {
				return err
			}
		}
	}

	// Outbox-событие об обновлении
	payload, _ := json.Marshal(map[string]interface{}{
		"task_id":    task.ID,
		"new_status": task.Status,
	})
	outbox := &models.Outbox{
		AggregateID: task.ID,
		EventType:   "task_updated",
		Payload:     payload,
	}
	if err := r.outboxRepo.SaveInTransaction(ctx, tx, outbox); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete – soft delete + outbox в транзакции
func (r *taskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `UPDATE tasks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	cmd, err := tx.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	// Outbox-событие об удалении
	payload, _ := json.Marshal(map[string]interface{}{
		"task_id": id,
	})
	outbox := &models.Outbox{
		AggregateID: id,
		EventType:   "task_deleted",
		Payload:     payload,
	}
	if err := r.outboxRepo.SaveInTransaction(ctx, tx, outbox); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
