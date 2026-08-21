package handlers

import (
	"encoding/json"
	"net/http"

	"taskflow/internal/analytics"
	"taskflow/internal/hub"
	"taskflow/internal/models"
	"taskflow/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TaskHandler struct {
	service   *services.TaskService
	hub       *hub.Hub
	analytics *analytics.ClickHouseAnalyticsClient
}

func NewTaskHandler(service *services.TaskService, h *hub.Hub, analytics *analytics.ClickHouseAnalyticsClient) *TaskHandler {
	return &TaskHandler{
		service:   service,
		hub:       h,
		analytics: analytics,
	}
}

// Create создаёт новую задачу
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

	if h.analytics != nil {
		h.analytics.SendEvent(analytics.Event{
			TaskID:    task.ID,
			EventType: "created",
			Status:    task.Status,
			Assignee:  task.Assignee,
		})
	}

	event := hub.WebSocketEvent{
		Type: "task_created",
		Data: task,
	}
	if data, err := json.Marshal(event); err == nil {
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

// Update обновляет задачу
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

	if h.analytics != nil {
		h.analytics.SendEvent(analytics.Event{
			TaskID:    task.ID,
			EventType: "updated",
			Status:    task.Status,
			Assignee:  task.Assignee,
		})
	}

	// Отправка WebSocket-события
	event := hub.WebSocketEvent{
		Type: "task_updated",
		Data: task,
	}
	if data, err := json.Marshal(event); err == nil {
		h.hub.Broadcast(data)
	}

	c.JSON(http.StatusOK, task)
}

// Delete мягко удаляет задачу
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

	if h.analytics != nil {
		h.analytics.SendEvent(analytics.Event{
			TaskID:    id,
			EventType: "deleted",
			Status:    "", // можно оставить пустым
			Assignee:  "",
		})
	}

	event := hub.WebSocketEvent{
		Type: "task_deleted",
		Data: map[string]interface{}{"id": id},
	}
	if data, err := json.Marshal(event); err == nil {
		h.hub.Broadcast(data)
	}

	c.Status(http.StatusNoContent)
}

// GetStats возвращает аналитику по задачам за последние 24 часа
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
