package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context values set by this middleware.
type contextKey string

const (
	// CtxUserID is the key under which the authenticated user's ID is stored.
	CtxUserID contextKey = "user_id"
)

// Claims represents the JWT token payload.
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// Auth returns a JWT auth middleware.
// It validates the Bearer token, extracts user ID, stores it in gin.Context,
// and aborts with 401 on failure.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header must be Bearer token"})
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Store user ID in context
		c.Set(string(CtxUserID), claims.UserID)
		c.Next()
	}
}

// CurrentUserID is a helper for handlers to pull the authenticated user ID.
func CurrentUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get(string(CtxUserID))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return 0, false
	}
	id, ok := v.(int64)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "bad auth context"})
		return 0, false
	}
	return id, true
}

// GenerateToken creates a new JWT token for the given user ID.
// Returns the token string, expiration time, and any error.
func GenerateToken(userID int64, secret string, expirationHours int) (string, time.Time, error) {
	expirationTime := time.Now().Add(time.Duration(expirationHours) * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   string(rune(userID)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}
