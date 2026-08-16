package service

import (
	"context"
	"errors"

	"task_tracker/internal/model"
)

// CommentService handles task comments.
type CommentService struct {
	commentRepo commentStore
	taskRepo    taskStore
	teamRepo    teamStore
}

// NewCommentService creates a new CommentService.
func NewCommentService(
	commentRepo commentStore,
	taskRepo taskStore,
	teamRepo teamStore,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		taskRepo:    taskRepo,
		teamRepo:    teamRepo,
	}
}

// CreateComment adds a comment to a task.
func (s *CommentService) CreateComment(ctx context.Context, taskID int64, input model.CreateCommentInput, userID int64) (*model.TaskComment, error) {
	// Get task to check team membership
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	// Verify user is a member of the team
	isMember, err := s.teamRepo.IsUserMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this team")
	}

	comment := &model.TaskComment{
		TaskID:  taskID,
		UserID:  userID,
		Content: input.Content,
	}

	if err := s.commentRepo.AddComment(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// ListComments retrieves all comments for a task.
func (s *CommentService) ListComments(ctx context.Context, taskID int64, userID int64) ([]model.TaskComment, error) {
	// Get task to check team membership
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	// Verify user is a member of the team
	isMember, err := s.teamRepo.IsUserMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this team")
	}

	return s.commentRepo.ListComments(ctx, taskID)
}
