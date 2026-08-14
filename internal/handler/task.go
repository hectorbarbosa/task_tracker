package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task_tracker/internal/middleware"
	"task_tracker/internal/model"
	"task_tracker/internal/service"
)

// TaskHandler handles task CRUD and history.
type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// Create godoc
// @Summary      Create a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        input body model.CreateTaskInput true "task data"
// @Success      201 {object} model.Task
// @Failure      400 {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/tasks [post]
func (h *TaskHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var input model.CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// List godoc
// @Summary      List tasks with filters and pagination
// @Tags         tasks
// @Produce      json
// @Param        team_id     query int    true "team id"
// @Param        status      query string false "status filter"
// @Param        assignee_id query int    false "assignee filter"
// @Param        limit       query int    false "page size"
// @Param        offset      query int    false "offset"
// @Success      200 {array} model.Task
// @Security     BearerAuth
// @Router       /api/v1/tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var filter model.TaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, err := h.taskService.ListTasks(c.Request.Context(), filter, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []model.Task{}
	}

	c.JSON(http.StatusOK, tasks)
}

// Update godoc
// @Summary      Update a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id path int true "task id"
// @Param        input body model.UpdateTaskInput true "update data"
// @Success      200 {object} model.Task
// @Failure      403 {object} object{error=string}
// @Failure      409 {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id} [put]
func (h *TaskHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var input model.UpdateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, input, userID)
	if err != nil {
		if err.Error() == "version mismatch: task was updated by another user" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "you do not have permission to edit this task" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// History godoc
// @Summary      Get task change history
// @Tags         tasks
// @Produce      json
// @Param        id path int true "task id"
// @Success      200 {array} model.TaskHistory
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id}/history [get]
func (h *TaskHandler) History(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	history, err := h.taskService.ListTaskHistory(c.Request.Context(), taskID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if history == nil {
		history = []model.TaskHistory{}
	}

	c.JSON(http.StatusOK, history)
}
