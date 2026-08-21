package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"taskflow/internal/analytics"
	"taskflow/internal/hub"
	"taskflow/internal/kafka"
	"taskflow/internal/models"
	"taskflow/internal/services"
)

type TaskHandler struct {
	service       *services.TaskService
	hub           *hub.Hub
	analytics     *analytics.ClickHouseAnalyticsClient
	kafkaProducer *kafka.Producer
}

func NewTaskHandler(
	service *services.TaskService,
	h *hub.Hub,
	analytics *analytics.ClickHouseAnalyticsClient,
	kafkaProducer *kafka.Producer,
) *TaskHandler {
	return &TaskHandler{
		service:       service,
		hub:           h,
		analytics:     analytics,
		kafkaProducer: kafkaProducer,
	}
}

// Create создаёт задачу и отправляет событие в Kafka
func (h *TaskHandler) Create(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Отправляем в Kafka
	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"task_id":    task.ID.String(),
			"event_type": "created",
			"status":     task.Status,
			"assignee":   task.Assignee,
		}
		_ = h.kafkaProducer.SendEvent(task.ID.String(), event)
	}

	// WebSocket
	wsEvent := hub.WebSocketEvent{
		Type: "task_created",
		Data: task,
	}
	if data, err := json.Marshal(wsEvent); err == nil {
		h.hub.Broadcast(data)
	}

	c.JSON(http.StatusCreated, task)
}

// Get возвращает задачу по ID
func (h *TaskHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	task, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// List возвращает список задач с фильтрацией и пагинацией
func (h *TaskHandler) List(c *gin.Context) {
	var filter models.TaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tasks, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks, "total": total})
}

// Update обновляет задачу и отправляет событие в Kafka
func (h *TaskHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == "task was modified concurrently or not found" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"task_id":    task.ID.String(),
			"event_type": "updated",
			"status":     task.Status,
			"assignee":   task.Assignee,
		}
		_ = h.kafkaProducer.SendEvent(task.ID.String(), event)
	}

	wsEvent := hub.WebSocketEvent{
		Type: "task_updated",
		Data: task,
	}
	if data, err := json.Marshal(wsEvent); err == nil {
		h.hub.Broadcast(data)
	}

	c.JSON(http.StatusOK, task)
}

// Delete удаляет задачу и отправляет событие в Kafka
func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"task_id":    id.String(),
			"event_type": "deleted",
			"status":     "",
			"assignee":   "",
		}
		_ = h.kafkaProducer.SendEvent(id.String(), event)
	}

	wsEvent := hub.WebSocketEvent{
		Type: "task_deleted",
		Data: map[string]interface{}{"id": id},
	}
	if data, err := json.Marshal(wsEvent); err == nil {
		h.hub.Broadcast(data)
	}

	c.Status(http.StatusNoContent)
}

// GetStats – аналитика из ClickHouse
func (h *TaskHandler) GetStats(c *gin.Context) {
	if h.analytics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics not available"})
		return
	}
	stats, err := h.analytics.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
