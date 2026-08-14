package repository

import (
	"context"
	"database/sql"
	"errors"

	"task_tracker/internal/model"
)

// TeamRepository provides access to team data in MySQL.
type TeamRepository struct {
	db *sql.DB
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// CreateTeam creates a new team and adds the creator as owner in a transaction.
func (r *TeamRepository) CreateTeam(ctx context.Context, team *model.Team, ownerID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert team
	query := `
		INSERT INTO teams (name, created_by)
		VALUES (?, ?)
	`
	result, err := tx.ExecContext(ctx, query, team.Name, ownerID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = id
	team.CreatedBy = ownerID

	// Add creator as owner
	memberQuery := `
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, memberQuery, team.ID, ownerID, model.RoleOwner)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ListUserTeams returns all teams where the user is a member.
func (r *TeamRepository) ListUserTeams(ctx context.Context, userID int64) ([]model.Team, error) {
	query := `
		SELECT t.id, t.name, t.created_by, t.created_at
		FROM teams t
		INNER JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		var team model.Team
		if err := rows.Scan(&team.ID, &team.Name, &team.CreatedBy, &team.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, rows.Err()
}

// GetTeamByID retrieves a team by ID.
func (r *TeamRepository) GetTeamByID(ctx context.Context, teamID int64) (*model.Team, error) {
	query := `
		SELECT id, name, created_by, created_at
		FROM teams
		WHERE id = ?
	`

	team := &model.Team{}
	err := r.db.QueryRowContext(ctx, query, teamID).Scan(
		&team.ID,
		&team.Name,
		&team.CreatedBy,
		&team.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return team, nil
}

// GetUserRole returns the user's role in a team.
func (r *TeamRepository) GetUserRole(ctx context.Context, teamID, userID int64) (model.Role, error) {
	query := `
		SELECT role
		FROM team_members
		WHERE team_id = ? AND user_id = ?
	`

	var role model.Role
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return role, nil
}

// AddTeamMember adds a user to a team with the specified role.
func (r *TeamRepository) AddTeamMember(ctx context.Context, teamID, userID int64, role model.Role) error {
	query := `
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, teamID, userID, role)
	return err
}

// UpdateMemberRole updates a member's role in a team.
func (r *TeamRepository) UpdateMemberRole(ctx context.Context, teamID, userID int64, role model.Role) error {
	query := `
		UPDATE team_members
		SET role = ?
		WHERE team_id = ? AND user_id = ?
	`
	_, err := r.db.ExecContext(ctx, query, role, teamID, userID)
	return err
}

// IsUserMember checks if a user is a member of a team.
func (r *TeamRepository) IsUserMember(ctx context.Context, teamID, userID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM team_members
		WHERE team_id = ? AND user_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
