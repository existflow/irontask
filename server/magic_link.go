package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
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

	// Check if user exists
	var userID string
	err := s.db.QueryRow(`SELECT id FROM users WHERE email = $1`, req.Email).Scan(&userID)
	if err != nil {
		// Don't reveal if email exists
		return c.JSON(http.StatusOK, map[string]string{"message": "if email exists, a magic link will be sent"})
	}

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.Logger().Error("token generation error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	token := hex.EncodeToString(tokenBytes)

	// Token expires in 15 minutes
	expiresAt := time.Now().Add(15 * time.Minute)

	// Insert magic link
	_, err = s.db.Exec(`
		INSERT INTO magic_links (email, token, expires_at)
		VALUES ($1, $2, $3)`,
		req.Email, token, expiresAt,
	)
	if err != nil {
		c.Logger().Error("db error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("Magic link created for: %s", req.Email)

	// In production, send email here
	return c.JSON(http.StatusOK, map[string]string{
		"message": "if email exists, a magic link will be sent",
		"token":   token, // Remove in production
	})
}

// handleMagicLinkVerify verifies a magic link and creates a session
func (s *Server) handleMagicLinkVerify(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
	}

	// Find magic link
	var email string
	var expiresAt time.Time
	var used bool
	err := s.db.QueryRow(`
		SELECT email, expires_at, used FROM magic_links 
		WHERE token = $1`,
		token,
	).Scan(&email, &expiresAt, &used)

	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token"})
	}

	if used {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token already used"})
	}

	if time.Now().After(expiresAt) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token expired"})
	}

	// Mark as used
	_, err = s.db.Exec(`UPDATE magic_links SET used = TRUE WHERE token = $1`, token)
	if err != nil {
		c.Logger().Error("db error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// Find user
	var userID string
	err = s.db.QueryRow(`SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Create session
	sessionToken, sessionExpires, err := s.createSession(userID)
	if err != nil {
		c.Logger().Error("session error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("Magic link login: %s", email)

	return c.JSON(http.StatusOK, authResponse{
		Token:     sessionToken,
		ExpiresAt: sessionExpires.Format(time.RFC3339),
		UserID:    userID,
	})
}
