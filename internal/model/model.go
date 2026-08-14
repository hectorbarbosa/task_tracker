package model

import (
	"encoding/json"
	"time"
)

// Role of a user inside a team.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// TaskStatus is the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// User is a registered account.
type User struct {
	ID           int64     `json:"id"         db:"id"`
	Email        string    `json:"email"      db:"email"`
	PasswordHash string    `json:"-"          db:"password_hash"` // never serialized
	Name         string    `json:"name"       db:"name"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Team is a collaboration unit owned by a user.
type Team struct {
	ID        int64     `json:"id"         db:"id"`
	Name      string    `json:"name"       db:"name"`
	CreatedBy int64     `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TeamMember binds a user to a team with a role.
type TeamMember struct {
	TeamID int64 `json:"team_id" db:"team_id"`
	UserID int64 `json:"user_id" db:"user_id"`
	Role   Role  `json:"role"    db:"role"`
}

// Task is a unit of work inside a team. Version is used for optimistic locking.
type Task struct {
	ID          int64       `json:"id"          db:"id"`
	TeamID      int64       `json:"team_id"     db:"team_id"`
	Title       string      `json:"title"       db:"title"`
	Description string      `json:"description" db:"description"`
	Status      TaskStatus  `json:"status"      db:"status"`
	CreatedBy   int64       `json:"created_by"  db:"created_by"`
	AssigneeID  *int64      `json:"assignee_id" db:"assignee_id"`
	CreatedAt   time.Time   `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"  db:"updated_at"`
	ClosedAt    *time.Time  `json:"closed_at"   db:"closed_at"`
	Version     int         `json:"version"     db:"version"`
}

// TaskHistory records one change to a task. Changes is a JSON object
// describing old/new values for the fields that changed.
type TaskHistory struct {
	ID        int64           `json:"id"         db:"id"`
	TaskID    int64           `json:"task_id"    db:"task_id"`
	ChangedBy int64           `json:"changed_by" db:"changed_by"`
	Changes   json.RawMessage `json:"changes"    db:"changes"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// TaskComment is a free-form comment left by a team member on a task.
type TaskComment struct {
	ID        int64     `json:"id"         db:"id"`
	TaskID    int64     `json:"task_id"    db:"task_id"`
	UserID    int64     `json:"user_id"    db:"user_id"`
	Content   string    `json:"content"    db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ─── Request / response DTOs ─────────────────────────────────────────────────
// Kept here so handlers can reference them without pulling the whole service layer.

type RegisterInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"     binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateTeamInput struct {
	Name string `json:"name" binding:"required"`
}

type InviteInput struct {
	Email string `json:"email" binding:"required,email"`
	Role  Role   `json:"role"  binding:"required,oneof=admin member"`
}

type CreateTaskInput struct {
	TeamID     int64      `json:"team_id"     binding:"required"`
	Title      string     `json:"title"       binding:"required"`
	Description string    `json:"description"`
	Status     TaskStatus `json:"status"      binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID *int64     `json:"assignee_id"`
}

type UpdateTaskInput struct {
	Title       *string     `json:"title"`
	Description *string     `json:"description"`
	Status      *TaskStatus `json:"status"      binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID  *int64      `json:"assignee_id"`
	Version     int         `json:"version"     binding:"required"` // optimistic lock
}

type TaskFilter struct {
	TeamID     int64       `form:"team_id"     binding:"required"`
	Status     *TaskStatus `form:"status"`
	AssigneeID *int64      `form:"assignee_id"`
	Limit      int         `form:"limit"       binding:"omitempty,min=1,max=100"`
	Offset     int         `form:"offset"      binding:"omitempty,min=0"`
}

type CreateCommentInput struct {
	Content string `json:"content" binding:"required"`
}

// TeamStats is the shape of the /teams/:team_id/stats response.
type TeamStats struct {
	ByStatus      map[TaskStatus]int `json:"by_status"`
	TopAssignees  []TopAssignee      `json:"top_assignees"`
	AvgCloseSec   float64            `json:"avg_close_seconds"`
	TotalComments int64              `json:"total_comments"`
}

type TopAssignee struct {
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	ClosedCount  int    `json:"closed_count"`
}
