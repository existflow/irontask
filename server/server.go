package server

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/tphuc/irontask/server/database"
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
	if err := s.migrate(); err != nil {
		return nil, err
	}

	// Setup Echo
	s.setupEcho()

	return s, nil
}

func (s *Server) setupEcho() {
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

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
