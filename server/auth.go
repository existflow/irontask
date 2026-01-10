package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	UserID    string `json:"user_id"`
}

// handleRegister handles user registration
func (s *Server) handleRegister(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Validate
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username, email, and password required"})
	}

	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Logger().Error("bcrypt error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// Insert user
	var userID string
	err = s.db.QueryRow(`
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id`,
		req.Username, req.Email, string(hash),
	).Scan(&userID)

	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return c.JSON(http.StatusConflict, map[string]string{"error": "username or email already exists"})
		}
		c.Logger().Error("db error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// Create session
	token, expiresAt, err := s.createSession(userID)
	if err != nil {
		c.Logger().Error("session error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("User registered: %s", req.Username)

	return c.JSON(http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		UserID:    userID,
	})
}

// handleLogin handles user login
func (s *Server) handleLogin(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Find user
	var userID, passwordHash string
	err := s.db.QueryRow(`
		SELECT id, password_hash FROM users WHERE username = $1`,
		req.Username,
	).Scan(&userID, &passwordHash)

	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	// Create session
	token, expiresAt, err := s.createSession(userID)
	if err != nil {
		c.Logger().Error("session error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("User logged in: %s", req.Username)

	return c.JSON(http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		UserID:    userID,
	})
}

// handleMe returns current user info
func (s *Server) handleMe(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var username, email string
	err := s.db.QueryRow(`
		SELECT username, email FROM users WHERE id = $1`,
		userID,
	).Scan(&username, &email)

	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id":       userID,
		"username": username,
		"email":    email,
	})
}

// createSession creates a new session for a user
func (s *Server) createSession(userID string) (string, time.Time, error) {
	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(tokenBytes)

	// Session expires in 30 days
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err := s.db.Exec(`
		INSERT INTO sessions (user_id, token, expires_at)
		VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)

	return token, expiresAt, err
}
