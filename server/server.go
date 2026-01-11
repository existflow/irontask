package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/existflow/irontask/internal/logger"
	"github.com/existflow/irontask/server/database"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

// Server is the sync server
type Server struct {
	db      *sql.DB
	queries *database.Queries
	echo    *echo.Echo
}

// New creates a new server
func New(dbURL string) (*Server, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &Server{
		db:      db,
		queries: database.New(db),
	}

	// Run migrations
	logger.Info("Running database migrations")
	if err := s.migrate(); err != nil {
		logger.Error("Migration failed", logger.F("error", err))
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	logger.Info("Database migrations completed successfully")

	// Setup Echo
	s.setupEcho()

	return s, nil
}

func (s *Server) setupEcho() {
	e := echo.New()
	e.HideBanner = true

	// Order matters: RequestID must come before logging
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Consolidated request logging - single line with all info
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			reqID := c.Response().Header().Get(echo.HeaderXRequestID)

			// Process request
			err := next(c)

			// Log single line with request + response info
			res := c.Response()
			duration := time.Since(start)

			// Choose log level based on status
			status := res.Status
			logFn := logger.Info
			if status >= 500 {
				logFn = logger.Error
			} else if status >= 400 {
				logFn = logger.Warn
			}

			logFn("HTTP",
				logger.F("id", reqID[:8]), // Short request ID
				logger.F("method", req.Method),
				logger.F("path", req.URL.Path),
				logger.F("status", status),
				logger.F("duration", duration.Round(time.Microsecond).String()),
				logger.F("size", res.Size))

			return err
		}
	})

	// Health check
	e.GET("/health", s.handleHealth)

	// API v1
	api := e.Group("/api/v1")

	// Auth endpoints (public)
	api.POST("/register", s.handleRegister)
	api.POST("/login", s.handleLogin)
	api.POST("/magic-link", s.handleMagicLink)
	api.GET("/magic-link/:token", s.handleMagicLinkVerify)

	// Protected endpoints
	protected := api.Group("")
	protected.Use(s.authMiddleware)
	protected.GET("/me", s.handleMe)
	protected.POST("/logout", s.handleLogout)
	protected.GET("/sync", s.handleSyncPull)
	protected.POST("/sync", s.handleSyncPush)
	protected.POST("/clear", s.handleClear)

	s.echo = e
}

// Close closes the database connection
func (s *Server) Close() error {
	return s.db.Close()
}

// Router returns the HTTP handler
func (s *Server) Router() http.Handler {
	return s.echo
}

// Start starts the server
func (s *Server) Start(addr string) error {
	return s.echo.Start(addr)
}

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
