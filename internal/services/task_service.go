package services

import (
	"context"
	"errors"
	"time"

	"taskflow/internal/models"
	"taskflow/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TaskService содержит бизнес-логику работы с задачами
type TaskService struct {
	repo repository.TaskRepository
}

// NewTaskService создаёт новый экземпляр сервиса
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// Create создаёт новую задачу
func (s *TaskService) Create(ctx context.Context, req models.CreateTaskRequest) (*models.Task, error) {
	task := &models.Task{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Status:      "new", // всегда начинаем с new
		Assignee:    req.Assignee,
		DueDate:     req.DueDate,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// GetByID возвращает задачу по ID
func (s *TaskService) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	return s.repo.GetByID(ctx, id)
}

// List возвращает список задач с фильтрацией и пагинацией
func (s *TaskService) List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return s.repo.List(ctx, filter)
}

// Update обновляет задачу, записывая историю изменений в одной транзакции
func (s *TaskService) Update(ctx context.Context, id uuid.UUID, req models.UpdateTaskRequest) (*models.Task, error) {
	// 1. Получаем текущую задачу
	oldTask, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Копируем старую задачу и подготавливаем историю
	updated := *oldTask
	history := []models.TaskHistory{}

	// Проверяем каждое поле на изменение
	if req.Title != nil && *req.Title != oldTask.Title {
		updated.Title = *req.Title
		history = append(history, models.TaskHistory{
			TaskID:    oldTask.ID,
			FieldName: "title",
			OldValue:  &oldTask.Title,
			NewValue:  req.Title,
		})
	}
	if req.Description != nil && *req.Description != oldTask.Description {
		updated.Description = *req.Description
		history = append(history, models.TaskHistory{
			TaskID:    oldTask.ID,
			FieldName: "description",
			OldValue:  &oldTask.Description,
			NewValue:  req.Description,
		})
	}
	if req.Status != nil && *req.Status != oldTask.Status {
		updated.Status = *req.Status
		history = append(history, models.TaskHistory{
			TaskID:    oldTask.ID,
			FieldName: "status",
			OldValue:  &oldTask.Status,
			NewValue:  req.Status,
		})
	}
	if req.Assignee != nil && *req.Assignee != oldTask.Assignee {
		updated.Assignee = *req.Assignee
		history = append(history, models.TaskHistory{
			TaskID:    oldTask.ID,
			FieldName: "assignee",
			OldValue:  &oldTask.Assignee,
			NewValue:  req.Assignee,
		})
	}
	if req.DueDate != nil && (oldTask.DueDate == nil || !req.DueDate.Equal(*oldTask.DueDate)) {
		updated.DueDate = req.DueDate
		oldVal := ""
		if oldTask.DueDate != nil {
			oldVal = oldTask.DueDate.Format(time.RFC3339)
		}
		newVal := ""
		if req.DueDate != nil {
			newVal = req.DueDate.Format(time.RFC3339)
		}
		history = append(history, models.TaskHistory{
			TaskID:    oldTask.ID,
			FieldName: "due_date",
			OldValue:  &oldVal,
			NewValue:  &newVal,
		})
	}

	// Если нет изменений, возвращаем текущую задачу без обновления
	if len(history) == 0 {
		return oldTask, nil
	}

	// 3. Выполняем обновление в репозитории (в транзакции)
	err = s.repo.Update(ctx, &updated, history)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("task was modified concurrently or not found")
		}
		return nil, err
	}

	// 4. Возвращаем обновлённую задачу (с новым version)
	return &updated, nil
}

// Delete выполняет мягкое удаление задачи
func (s *TaskService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
