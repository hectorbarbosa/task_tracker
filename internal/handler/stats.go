package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task_tracker/internal/middleware"
	"task_tracker/internal/service"
)

// StatsHandler serves the per-team analytics report.
type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// Get godoc
// @Summary      Get team statistics
// @Description  Returns task-by-status counts, top-3 assignees by closed tasks
// @Description  in last 30 days, average close time, and comment count for the team.
// @Tags         stats
// @Produce      json
// @Param        team_id path int true "team id"
// @Success      200 {object} model.TeamStats
// @Failure      403 {object} object{error=string}
// @Security     BearerAuth
// @Router       /api/v1/teams/{team_id}/stats [get]
func (h *StatsHandler) Get(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	teamID, err := strconv.ParseInt(c.Param("team_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	stats, err := h.statsService.GetTeamStats(c.Request.Context(), teamID, userID)
	if err != nil {
		if err.Error() == "only owner or admin can view team statistics" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
