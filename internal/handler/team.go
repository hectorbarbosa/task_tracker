package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task_tracker/internal/middleware"
	"task_tracker/internal/model"
	"task_tracker/internal/service"
)

// TeamHandler handles team CRUD and invitations.
type TeamHandler struct {
	teamService *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}

// CreateTeam godoc
// @Summary      Create a team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        input body model.CreateTeamInput true "team data"
// @Success      201 {object} model.Team
// @Failure      400 {object} object{error=string}
// @Security     BearerAuth
// @Router       /teams [post]
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var input model.CreateTeamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.teamService.CreateTeam(c.Request.Context(), input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, team)
}

// ListTeams godoc
// @Summary      List current user's teams
// @Tags         teams
// @Produce      json
// @Success      200 {array} model.Team
// @Security     BearerAuth
// @Router       /teams [get]
func (h *TeamHandler) ListTeams(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	teams, err := h.teamService.ListUserTeams(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if teams == nil {
		teams = []model.Team{}
	}

	c.JSON(http.StatusOK, teams)
}

// Invite godoc
// @Summary      Invite a user to a team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        id path int true "team id"
// @Param        input body model.InviteInput true "invite data"
// @Success      200 {object} model.TeamMember
// @Failure      403 {object} object{error=string}
// @Security     BearerAuth
// @Router       /teams/{id}/invite [post]
func (h *TeamHandler) Invite(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var input model.InviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.teamService.InviteUser(c.Request.Context(), teamID, userID, input)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, member)
}
