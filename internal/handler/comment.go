package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task_tracker/internal/middleware"
	"task_tracker/internal/model"
	"task_tracker/internal/service"
)

// CommentHandler handles task comments.
type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// Create godoc
// @Summary      Add a comment to a task
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        id path int true "task id"
// @Param        input body model.CreateCommentInput true "comment data"
// @Success      201 {object} model.TaskComment
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var input model.CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), taskID, input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// List godoc
// @Summary      List comments for a task
// @Tags         comments
// @Produce      json
// @Param        id path int true "task id"
// @Success      200 {array} model.TaskComment
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id}/comments [get]
func (h *CommentHandler) List(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	comments, err := h.commentService.ListComments(c.Request.Context(), taskID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if comments == nil {
		comments = []model.TaskComment{}
	}

	c.JSON(http.StatusOK, comments)
}
