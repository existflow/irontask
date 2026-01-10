package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/existflow/irontask/server/database"
)

type magicLinkRequest struct {
	Email string `json:"email"`
}

// handleMagicLink creates a magic link for passwordless login
func (s *Server) handleMagicLink(c echo.Context) error {
	var req magicLinkRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
	}

	// Generate magic link token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.Logger().Error("token generation error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	token := hex.EncodeToString(tokenBytes)

	var email string
	// Check if user exists
	user, err := s.queries.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Auto-register user
			username := req.Email
			if parts := strings.Split(req.Email, "@"); len(parts) > 0 {
				username = parts[0]
			}

			fmt.Printf("🌱 Auto-registering user: %s (%s)\n", username, req.Email)

			newUser, err := s.queries.CreateUser(c.Request().Context(), database.CreateUserParams{
				Username:     username,
				Email:        req.Email,
				PasswordHash: "MAGIC_LINK_ONLY_" + token[:16],
			})
			if err != nil {
				c.Logger().Error("auto-registration error:", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to auto-register"})
			}
			email = newUser.Email
		} else {
			c.Logger().Error("db error:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	} else {
		email = user.Email
	}

	// Token expires in 15 minutes
	expiresAt := time.Now().Add(15 * time.Minute)

	// Insert magic link
	err = s.queries.CreateMagicLink(c.Request().Context(), database.CreateMagicLinkParams{
		Email:     email,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		c.Logger().Error("db error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	fmt.Printf("\n✨ MAGIC LINK GENERATED ✨\nEmail: %s\nToken: %s\n\n", req.Email, token)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "if email exists, a magic link will be sent",
		"token":   token,
	})
}

// handleMagicLinkVerify verifies a magic link and creates a session
func (s *Server) handleMagicLinkVerify(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
	}

	// Find magic link
	link, err := s.queries.GetMagicLink(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token"})
	}

	if link.Used.Bool {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token already used"})
	}

	if time.Now().After(link.ExpiresAt) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token expired"})
	}

	// Mark as used
	if err := s.queries.MarkMagicLinkUsed(context.Background(), token); err != nil {
		c.Logger().Error("db error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// Find user
	user, err := s.queries.GetUserByEmail(c.Request().Context(), link.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Create session
	sessionToken, sessionExpires, err := s.createSession(user.ID.String())
	if err != nil {
		c.Logger().Error("session error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("Magic link login: %s", link.Email)

	return c.JSON(http.StatusOK, authResponse{
		Token:     sessionToken,
		ExpiresAt: sessionExpires.Format(time.RFC3339),
		UserID:    user.ID.String(),
	})
}
