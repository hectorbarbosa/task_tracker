package service

import (
	"context"
	"errors"
	"time"

	"task_tracker/internal/model"
	"task_tracker/internal/repository"
)

// TaskService handles task CRUD and history.
type TaskService struct {
	taskRepo  *repository.TaskRepository
	teamRepo  *repository.TeamRepository
	cacheRepo *repository.CacheRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	taskRepo *repository.TaskRepository,
	teamRepo *repository.TeamRepository,
	cacheRepo *repository.CacheRepository,
) *TaskService {
	return &TaskService{
		taskRepo:  taskRepo,
		teamRepo:  teamRepo,
		cacheRepo: cacheRepo,
	}
}

// CreateTask creates a new task in a team.
func (s *TaskService) CreateTask(ctx context.Context, input model.CreateTaskInput, userID int64) (*model.Task, error) {
	// Verify user is a member of the team
	isMember, err := s.teamRepo.IsUserMember(ctx, input.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this team")
	}

	// If assignee is specified, verify they are a member of the team
	if input.AssigneeID != nil {
		isAssigneeMember, err := s.teamRepo.IsUserMember(ctx, input.TeamID, *input.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !isAssigneeMember {
			return nil, errors.New("assignee must be a member of the team")
		}
	}

	task := &model.Task{
		TeamID:      input.TeamID,
		Title:       input.Title,
		Description: input.Description,
		Status:      model.TaskStatusTodo,
		CreatedBy:   userID,
		AssigneeID:  input.AssigneeID,
	}

	if input.Status != "" {
		task.Status = input.Status
	}

	if err := s.taskRepo.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	// Invalidate cache for this team
	if err := s.cacheRepo.InvalidateTeamTasks(ctx, input.TeamID); err != nil {
		// Log error but don't fail
	}

	return task, nil
}

// ListTasks retrieves tasks with filters.
func (s *TaskService) ListTasks(ctx context.Context, filter model.TaskFilter, userID int64) ([]model.Task, error) {
	// Verify user is a member of the team
	isMember, err := s.teamRepo.IsUserMember(ctx, filter.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this team")
	}

	// Try to get from cache first
	cachedTasks, err := s.cacheRepo.GetTaskList(ctx, filter)
	if err != nil {
		// Log error but continue with DB query
		// Cache errors should not break functionality
	} else if cachedTasks != nil {
		// Cache hit
		return cachedTasks, nil
	}

	// Cache miss - query database
	tasks, err := s.taskRepo.ListTasks(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors)
	if err := s.cacheRepo.SetTaskList(ctx, filter, tasks); err != nil {
		// Log error but don't fail
	}

	return tasks, nil
}

// UpdateTask updates a task with permission checks and history recording.
func (s *TaskService) UpdateTask(ctx context.Context, taskID int64, input model.UpdateTaskInput, userID int64) (*model.Task, error) {
	// Get current task
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	// Get user's role in the team
	role, err := s.teamRepo.GetUserRole(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, errors.New("you are not a member of this team")
	}

	// Check permissions
	canEdit := false
	if role == model.RoleOwner || role == model.RoleAdmin {
		// Owner and admin can edit any task
		canEdit = true
	} else if task.CreatedBy == userID {
		// Creator can edit their own task
		canEdit = true
	} else if task.AssigneeID != nil && *task.AssigneeID == userID {
		// Assignee can only change status
		if input.Title != nil || input.Description != nil || input.AssigneeID != nil {
			return nil, errors.New("assignee can only change status")
		}
		canEdit = true
	}

	if !canEdit {
		return nil, errors.New("you do not have permission to edit this task")
	}

	// Build changes map for history
	changes := make(map[string]interface{})

	// Apply updates
	if input.Title != nil {
		changes["title"] = map[string]interface{}{
			"old": task.Title,
			"new": *input.Title,
		}
		task.Title = *input.Title
	}

	if input.Description != nil {
		changes["description"] = map[string]interface{}{
			"old": task.Description,
			"new": *input.Description,
		}
		task.Description = *input.Description
	}

	if input.Status != nil {
		changes["status"] = map[string]interface{}{
			"old": task.Status,
			"new": *input.Status,
		}
		task.Status = *input.Status

		// Set closed_at if status changed to done
		if *input.Status == model.TaskStatusDone && task.ClosedAt == nil {
			now := time.Now()
			task.ClosedAt = &now
			changes["closed_at"] = map[string]interface{}{
				"old": nil,
				"new": now,
			}
		} else if *input.Status != model.TaskStatusDone && task.ClosedAt != nil {
			changes["closed_at"] = map[string]interface{}{
				"old": task.ClosedAt,
				"new": nil,
			}
			task.ClosedAt = nil
		}
	}

	if input.AssigneeID != nil {
		// Verify new assignee is a member of the team
		if *input.AssigneeID != 0 { // 0 means unassign
			isAssigneeMember, err := s.teamRepo.IsUserMember(ctx, task.TeamID, *input.AssigneeID)
			if err != nil {
				return nil, err
			}
			if !isAssigneeMember {
				return nil, errors.New("assignee must be a member of the team")
			}
		}

		var oldAssignee interface{}
		if task.AssigneeID != nil {
			oldAssignee = *task.AssigneeID
		}

		changes["assignee_id"] = map[string]interface{}{
			"old": oldAssignee,
			"new": *input.AssigneeID,
		}

		if *input.AssigneeID == 0 {
			task.AssigneeID = nil
		} else {
			task.AssigneeID = input.AssigneeID
		}
	}

	// Check version for optimistic locking
	if input.Version != task.Version {
		return nil, repository.ErrVersionMismatch
	}

	// Update task with history
	if err := s.taskRepo.UpdateTask(ctx, task, changes); err != nil {
		return nil, err
	}

	// Invalidate cache for this team
	if err := s.cacheRepo.InvalidateTeamTasks(ctx, task.TeamID); err != nil {
		// Log error but don't fail
	}

	return task, nil
}

// ListTaskHistory retrieves the change history for a task.
func (s *TaskService) ListTaskHistory(ctx context.Context, taskID int64, userID int64) ([]model.TaskHistory, error) {
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

	return s.taskRepo.ListTaskHistory(ctx, taskID)
}
