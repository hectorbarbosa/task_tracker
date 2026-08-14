package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"task_tracker/internal/model"
)

// TaskRepository provides access to task data in MySQL.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// CreateTask creates a new task in the database.
func (r *TaskRepository) CreateTask(ctx context.Context, task *model.Task) error {
	query := `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		task.TeamID,
		task.Title,
		task.Description,
		task.Status,
		task.CreatedBy,
		task.AssigneeID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	task.ID = id
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	task.Version = 1

	return nil
}

// GetTaskByID retrieves a task by ID.
func (r *TaskRepository) GetTaskByID(ctx context.Context, taskID int64) (*model.Task, error) {
	query := `
		SELECT id, team_id, title, description, status, created_by, assignee_id,
		       created_at, updated_at, closed_at, version
		FROM tasks
		WHERE id = ?
	`

	task := &model.Task{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID,
		&task.TeamID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedBy,
		&task.AssigneeID,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.ClosedAt,
		&task.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return task, nil
}

// ListTasks retrieves tasks with filters and pagination.
func (r *TaskRepository) ListTasks(ctx context.Context, filter model.TaskFilter) ([]model.Task, error) {
	query := `
		SELECT id, team_id, title, description, status, created_by, assignee_id,
		       created_at, updated_at, closed_at, version
		FROM tasks
		WHERE team_id = ?
	`
	args := []interface{}{filter.TeamID}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}

	if filter.AssigneeID != nil {
		query += " AND assignee_id = ?"
		args = append(args, *filter.AssigneeID)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 20" // default limit
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(
			&task.ID,
			&task.TeamID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedBy,
			&task.AssigneeID,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.ClosedAt,
			&task.Version,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTask updates a task with optimistic locking.
// Returns ErrVersionMismatch if version doesn't match.
var ErrVersionMismatch = errors.New("version mismatch: task was updated by another user")

func (r *TaskRepository) UpdateTask(ctx context.Context, task *model.Task, changes map[string]interface{}) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update task with version check
	query := `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?,
		    closed_at = ?, version = version + 1
		WHERE id = ? AND version = ?
	`

	var closedAt interface{}
	if task.ClosedAt != nil {
		closedAt = *task.ClosedAt
	}

	result, err := tx.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Status,
		task.AssigneeID,
		closedAt,
		task.ID,
		task.Version,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrVersionMismatch
	}

	// Record history
	if len(changes) > 0 {
		changesJSON, err := json.Marshal(changes)
		if err != nil {
			return err
		}

		historyQuery := `
			INSERT INTO task_history (task_id, changed_by, changes)
			VALUES (?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, historyQuery,
			task.ID,
			task.CreatedBy, // This should be the user making the change
			changesJSON,
		)
		if err != nil {
			return err
		}
	}

	task.Version++
	task.UpdatedAt = time.Now()

	return tx.Commit()
}

// AddComment adds a comment to a task.
func (r *TaskRepository) AddComment(ctx context.Context, comment *model.TaskComment) error {
	query := `
		INSERT INTO task_comments (task_id, user_id, content)
		VALUES (?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		comment.TaskID,
		comment.UserID,
		comment.Content,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	comment.ID = id
	comment.CreatedAt = time.Now()

	return nil
}

// ListComments retrieves all comments for a task.
func (r *TaskRepository) ListComments(ctx context.Context, taskID int64) ([]model.TaskComment, error) {
	query := `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.TaskComment
	for rows.Next() {
		var comment model.TaskComment
		if err := rows.Scan(
			&comment.ID,
			&comment.TaskID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, rows.Err()
}

// ListTaskHistory retrieves the change history for a task.
func (r *TaskRepository) ListTaskHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	query := `
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []model.TaskHistory
	for rows.Next() {
		var h model.TaskHistory
		if err := rows.Scan(
			&h.ID,
			&h.TaskID,
			&h.ChangedBy,
			&h.Changes,
			&h.CreatedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}
