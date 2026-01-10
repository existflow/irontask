package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/existflow/irontask/server/database"
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
	user, err := s.queries.CreateUser(c.Request().Context(), database.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	userID := user.ID.String()

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
	user, err := s.queries.GetUserByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	// Create session
	token, expiresAt, err := s.createSession(user.ID.String())
	if err != nil {
		c.Logger().Error("session error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	c.Logger().Infof("User logged in: %s", req.Username)

	return c.JSON(http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		UserID:    user.ID.String(),
	})
}

// handleMe returns current user info
func (s *Server) handleMe(c echo.Context) error {
	userIDStr := c.Get("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	user, err := s.queries.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id":       user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
	})
}

// handleLogout revokes the current session
func (s *Server) handleLogout(c echo.Context) error {
	// Token is already validated by middleware if this is protected
	// But we need to extract it again.
	// We can get it from header manually.
	auth := c.Request().Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token"})
	}

	if err := s.queries.DeleteSession(c.Request().Context(), token); err != nil {
		c.Logger().Error("logout error:", err)
		// Even if error, we probably want to say success to client?
		// But 500 is safer if DB failed.
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
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

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", time.Time{}, err
	}

	token, err = s.queries.CreateSession(context.Background(), database.CreateSessionParams{
		UserID:    userUUID,
		Token:     token,
		ExpiresAt: expiresAt,
	})

	return token, expiresAt, err
}
