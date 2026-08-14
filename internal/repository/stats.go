package repository

import (
	"context"
	"database/sql"

	"task_tracker/internal/model"
)

// StatsRepository provides SQL analytics for teams.
type StatsRepository struct {
	db *sql.DB
}

// NewStatsRepository creates a new StatsRepository.
func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// GetTeamStats returns comprehensive statistics for a team using efficient queries.
// Uses 4 separate queries (no N+1) for reliability and clarity.
func (r *StatsRepository) GetTeamStats(ctx context.Context, teamID int64) (*model.TeamStats, error) {
	stats := &model.TeamStats{
		ByStatus: make(map[model.TaskStatus]int),
	}

	// 1. Tasks by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE team_id = ?
		GROUP BY status
	`
	rows, err := r.db.QueryContext(ctx, statusQuery, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status model.TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Top 3 assignees by closed tasks in last 30 days
	topAssigneesQuery := `
		SELECT
			t.assignee_id,
			u.name,
			COUNT(*) as closed_count
		FROM tasks t
		INNER JOIN users u ON t.assignee_id = u.id
		WHERE t.team_id = ?
		  AND t.status = 'done'
		  AND t.closed_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		  AND t.assignee_id IS NOT NULL
		GROUP BY t.assignee_id, u.name
		ORDER BY closed_count DESC
		LIMIT 3
	`
	rows2, err := r.db.QueryContext(ctx, topAssigneesQuery, teamID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	stats.TopAssignees = []model.TopAssignee{}
	for rows2.Next() {
		var assignee model.TopAssignee
		if err := rows2.Scan(&assignee.UserID, &assignee.Name, &assignee.ClosedCount); err != nil {
			return nil, err
		}
		stats.TopAssignees = append(stats.TopAssignees, assignee)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	// 3. Average close time in seconds
	avgCloseQuery := `
		SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) as avg_seconds
		FROM tasks
		WHERE team_id = ?
		  AND status = 'done'
		  AND closed_at IS NOT NULL
	`
	var avgCloseSec sql.NullFloat64
	err = r.db.QueryRowContext(ctx, avgCloseQuery, teamID).Scan(&avgCloseSec)
	if err != nil {
		return nil, err
	}
	if avgCloseSec.Valid {
		stats.AvgCloseSec = avgCloseSec.Float64
	}

	// 4. Total comments for team tasks
	commentsQuery := `
		SELECT COUNT(*) as total_comments
		FROM task_comments tc
		INNER JOIN tasks t ON tc.task_id = t.id
		WHERE t.team_id = ?
	`
	var totalComments int64
	err = r.db.QueryRowContext(ctx, commentsQuery, teamID).Scan(&totalComments)
	if err != nil {
		return nil, err
	}
	stats.TotalComments = totalComments

	return stats, nil
}

// getTeamStatsSimple uses multiple efficient queries (no N+1) for better reliability.
func (r *StatsRepository) getTeamStatsSimple(ctx context.Context, teamID int64) (*model.TeamStats, error) {
	stats := &model.TeamStats{
		ByStatus: make(map[model.TaskStatus]int),
	}

	// 1. Tasks by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE team_id = ?
		GROUP BY status
	`
	rows, err := r.db.QueryContext(ctx, statusQuery, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status model.TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Top 3 assignees by closed tasks in last 30 days
	topAssigneesQuery := `
		SELECT
			t.assignee_id,
			u.name,
			COUNT(*) as closed_count
		FROM tasks t
		INNER JOIN users u ON t.assignee_id = u.id
		WHERE t.team_id = ?
		  AND t.status = 'done'
		  AND t.closed_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		  AND t.assignee_id IS NOT NULL
		GROUP BY t.assignee_id, u.name
		ORDER BY closed_count DESC
		LIMIT 3
	`
	rows2, err := r.db.QueryContext(ctx, topAssigneesQuery, teamID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	stats.TopAssignees = []model.TopAssignee{}
	for rows2.Next() {
		var assignee model.TopAssignee
		if err := rows2.Scan(&assignee.UserID, &assignee.Name, &assignee.ClosedCount); err != nil {
			return nil, err
		}
		stats.TopAssignees = append(stats.TopAssignees, assignee)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	// 3. Average close time in seconds
	avgCloseQuery := `
		SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) as avg_seconds
		FROM tasks
		WHERE team_id = ?
		  AND status = 'done'
		  AND closed_at IS NOT NULL
	`
	var avgCloseSec sql.NullFloat64
	err = r.db.QueryRowContext(ctx, avgCloseQuery, teamID).Scan(&avgCloseSec)
	if err != nil {
		return nil, err
	}
	if avgCloseSec.Valid {
		stats.AvgCloseSec = avgCloseSec.Float64
	}

	// 4. Total comments for team tasks
	commentsQuery := `
		SELECT COUNT(*) as total_comments
		FROM task_comments tc
		INNER JOIN tasks t ON tc.task_id = t.id
		WHERE t.team_id = ?
	`
	var totalComments int64
	err = r.db.QueryRowContext(ctx, commentsQuery, teamID).Scan(&totalComments)
	if err != nil {
		return nil, err
	}
	stats.TotalComments = totalComments

	return stats, nil
}
