package service

import (
	"context"
	"errors"

	"task_tracker/internal/model"
)

// StatsService handles team analytics.
type StatsService struct {
	statsRepo statsStore
	teamRepo  teamStore
}

// NewStatsService creates a new StatsService.
func NewStatsService(statsRepo statsStore, teamRepo teamStore) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		teamRepo:  teamRepo,
	}
}

// GetTeamStats returns statistics for a team.
// Only owner or admin can access team statistics.
func (s *StatsService) GetTeamStats(ctx context.Context, teamID int64, userID int64) (*model.TeamStats, error) {
	// Verify team exists
	team, err := s.teamRepo.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, errors.New("team not found")
	}

	// Get user's role in the team
	role, err := s.teamRepo.GetUserRole(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, errors.New("you are not a member of this team")
	}

	// Check permissions: only owner or admin can view stats
	if role != model.RoleOwner && role != model.RoleAdmin {
		return nil, errors.New("only owner or admin can view team statistics")
	}

	return s.statsRepo.GetTeamStats(ctx, teamID)
}
