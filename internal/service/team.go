package service

import (
	"context"
	"errors"

	"task_tracker/internal/model"
)

// TeamService handles team management and invitations.
type TeamService struct {
	teamRepo teamStore
	userRepo userStore
}

// NewTeamService creates a new TeamService.
func NewTeamService(teamRepo teamStore, userRepo userStore) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// CreateTeam creates a new team with the current user as owner.
func (s *TeamService) CreateTeam(ctx context.Context, input model.CreateTeamInput, userID int64) (*model.Team, error) {
	team := &model.Team{
		Name: input.Name,
	}

	if err := s.teamRepo.CreateTeam(ctx, team, userID); err != nil {
		return nil, err
	}

	return team, nil
}

// ListUserTeams returns all teams where the user is a member.
func (s *TeamService) ListUserTeams(ctx context.Context, userID int64) ([]model.Team, error) {
	return s.teamRepo.ListUserTeams(ctx, userID)
}

// InviteUser adds a user to a team with the specified role.
// Only owner or admin can invite users.
func (s *TeamService) InviteUser(ctx context.Context, teamID, inviterID int64, input model.InviteInput) (*model.TeamMember, error) {
	// Verify team exists
	team, err := s.teamRepo.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, errors.New("team not found")
	}

	// Verify inviter is a member
	inviterRole, err := s.teamRepo.GetUserRole(ctx, teamID, inviterID)
	if err != nil {
		return nil, err
	}
	if inviterRole == "" {
		return nil, errors.New("you are not a member of this team")
	}

	// Check permissions: only owner or admin can invite
	if inviterRole != model.RoleOwner && inviterRole != model.RoleAdmin {
		return nil, errors.New("only owner or admin can invite users")
	}

	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Prevent assigning owner role via invitation
	if input.Role == model.RoleOwner {
		return nil, errors.New("cannot assign owner role via invitation")
	}

	// Prevent admin from changing owner's role
	if inviterRole == model.RoleAdmin && team.CreatedBy == user.ID {
		return nil, errors.New("admin cannot modify owner")
	}

	// Add user to team atomically (checks membership and inserts in transaction)
	if err := s.teamRepo.InviteUser(ctx, teamID, user.ID, input.Role); err != nil {
		return nil, err
	}

	return &model.TeamMember{
		TeamID: teamID,
		UserID: user.ID,
		Role:   input.Role,
	}, nil
}

// GetUserRole returns the user's role in a team.
func (s *TeamService) GetUserRole(ctx context.Context, teamID, userID int64) (model.Role, error) {
	return s.teamRepo.GetUserRole(ctx, teamID, userID)
}

// IsUserMember checks if a user is a member of a team.
func (s *TeamService) IsUserMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return s.teamRepo.IsUserMember(ctx, teamID, userID)
}
