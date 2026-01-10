package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// authMiddleware checks for valid session token
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get token from Authorization header
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authorization required"})
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
		}

		// Validate session
		session, err := s.queries.GetSession(c.Request().Context(), token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		}

		if time.Now().After(session.ExpiresAt) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token expired"})
		}

		// Add user ID to context
		c.Set("user_id", session.UserID.String())
		return next(c)
	}
}
