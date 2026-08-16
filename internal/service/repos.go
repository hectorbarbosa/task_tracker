package service

import (
	"context"

	"task_tracker/internal/model"
)

// taskStore provides access to task data.
type taskStore interface {
	CreateTask(ctx context.Context, task *model.Task) error
	GetTaskByID(ctx context.Context, taskID int64) (*model.Task, error)
	ListTasks(ctx context.Context, filter model.TaskFilter) ([]model.Task, error)
	UpdateTask(ctx context.Context, task *model.Task, changes map[string]interface{}) error
	ListTaskHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error)
}

// teamStore provides access to team data and membership checks.
type teamStore interface {
	CreateTeam(ctx context.Context, team *model.Team, ownerID int64) error
	GetTeamByID(ctx context.Context, teamID int64) (*model.Team, error)
	ListUserTeams(ctx context.Context, userID int64) ([]model.Team, error)
	GetUserRole(ctx context.Context, teamID, userID int64) (model.Role, error)
	IsUserMember(ctx context.Context, teamID, userID int64) (bool, error)
	InviteUser(ctx context.Context, teamID, userID int64, role model.Role) error
}

// cacheStore provides caching for task lists.
type cacheStore interface {
	GetTaskList(ctx context.Context, filter model.TaskFilter) ([]model.Task, bool, error)
	SetTaskList(ctx context.Context, filter model.TaskFilter, tasks []model.Task) error
	InvalidateTeamTasks(ctx context.Context, teamID int64) error
}

// commentStore provides access to task comments.
type commentStore interface {
	AddComment(ctx context.Context, comment *model.TaskComment) error
	ListComments(ctx context.Context, taskID int64) ([]model.TaskComment, error)
}

// userStore provides access to user data.
type userStore interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

// statsStore provides SQL analytics for teams.
type statsStore interface {
	GetTeamStats(ctx context.Context, teamID int64) (*model.TeamStats, error)
}
