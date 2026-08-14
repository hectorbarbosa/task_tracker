package repository

import (
	"context"
	"database/sql"
	"time"

	"task_tracker/internal/model"
)

// CommentRepository provides access to comment data in MySQL.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// AddComment adds a comment to a task.
func (r *CommentRepository) AddComment(ctx context.Context, comment *model.TaskComment) error {
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
func (r *CommentRepository) ListComments(ctx context.Context, taskID int64) ([]model.TaskComment, error) {
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
