package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"taskflow/internal/cache"
	"taskflow/internal/models"

	"github.com/google/uuid"
)

// TaskRepositoryCache декорирует TaskRepository и добавляет кеширование
type TaskRepositoryCache struct {
	repo    TaskRepository
	cache   cache.Cache
	listTTL time.Duration
	itemTTL time.Duration
}

// NewTaskRepositoryCache создаёт новый экземпляр декоратора
func NewTaskRepositoryCache(repo TaskRepository, c cache.Cache) *TaskRepositoryCache {
	return &TaskRepositoryCache{
		repo:    repo,
		cache:   c,
		listTTL: 1 * time.Minute,
		itemTTL: 30 * time.Second,
	}
}

func (r *TaskRepositoryCache) Create(ctx context.Context, task *models.Task) error {
	err := r.repo.Create(ctx, task)
	if err == nil {
		_ = r.cache.DeletePattern(ctx, "tasks:list:*")
		log.Println("cache: invalidated list keys after Create")
	}
	return err
}

func (r *TaskRepositoryCache) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	key := fmt.Sprintf("task:%s", id.String())
	data, err := r.cache.Get(ctx, key)
	if err == nil && data != nil {
		var task models.Task
		if json.Unmarshal(data, &task) == nil {
			log.Println("cache: hit for task", id)
			return &task, nil
		}
	}
	log.Println("cache: miss for task", id)
	task, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(task); err == nil {
		_ = r.cache.Set(ctx, key, b, r.itemTTL)
	}
	return task, nil
}

func (r *TaskRepositoryCache) List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error) {
	filterBytes, _ := json.Marshal(filter)
	key := fmt.Sprintf("tasks:list:%s", string(filterBytes))
	data, err := r.cache.Get(ctx, key)
	if err == nil && data != nil {
		var result struct {
			Tasks []models.Task `json:"tasks"`
			Total int           `json:"total"`
		}
		if json.Unmarshal(data, &result) == nil {
			log.Println("cache: hit for list", key)
			return result.Tasks, result.Total, nil
		}
	}
	log.Println("cache: miss for list", key)
	tasks, total, err := r.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	result := struct {
		Tasks []models.Task `json:"tasks"`
		Total int           `json:"total"`
	}{tasks, total}
	if b, err := json.Marshal(result); err == nil {
		_ = r.cache.Set(ctx, key, b, r.listTTL)
	}
	return tasks, total, nil
}

func (r *TaskRepositoryCache) Update(ctx context.Context, task *models.Task, history []models.TaskHistory) error {
	err := r.repo.Update(ctx, task, history)
	if err == nil {
		_ = r.cache.Delete(ctx, fmt.Sprintf("task:%s", task.ID.String()))
		_ = r.cache.DeletePattern(ctx, "tasks:list:*")
		log.Println("cache: invalidated task and list keys after Update")
	}
	return err
}

func (r *TaskRepositoryCache) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.repo.Delete(ctx, id)
	if err == nil {
		_ = r.cache.Delete(ctx, fmt.Sprintf("task:%s", id.String()))
		_ = r.cache.DeletePattern(ctx, "tasks:list:*")
		log.Println("cache: invalidated task and list keys after Delete")
	}
	return err
}
