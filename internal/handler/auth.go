package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"task_tracker/internal/model"
	"task_tracker/internal/service"
)

// AuthHandler handles registration and login.
type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account and returns a JWT.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body model.RegisterInput true "registration data"
// @Success      201 {object} model.TokenResponse
// @Failure      400 {object} object{error=string}
// @Router       /api/v1/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input model.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenResp, err := h.authService.Register(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tokenResp)
}

// Login godoc
// @Summary      Authenticate a user
// @Description  Validates credentials and returns a JWT.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body model.LoginInput true "credentials"
// @Success      200 {object} model.TokenResponse
// @Failure      400 {object} object{error=string}
// @Failure      401 {object} object{error=string}
// @Router       /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input model.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenResp, err := h.authService.Login(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenResp)
}
