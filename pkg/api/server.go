package api

import (
	//"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	//"github.com/Ecook14/gocrewwai/pkg/config"
)

// Server represents the API and Streaming server.
type Server struct {
	router *gin.Engine
	port   int
}

// NewServer creates a new API server with CORS enabled.
func NewServer() *Server {
	// Standard Gin setup
	r := gin.Default()

	// Elite CORS configuration for React Flow Frontend
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"}, // React/Vite defaults
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	s := &Server{
		router: r,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/health", s.handleHealth)
		v1.POST("/crews/kickoff", s.handleKickoff)
		v1.GET("/sessions/:id", s.handleGetSession)
		v1.GET("/stream/:id", s.handleSSEStream)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "1.0.0-Stable",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// Run starts the server on the given address.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// ServeStatic enables the delivery of static files from an embedded filesystem.
func (s *Server) ServeStatic(fs http.FileSystem) {
	s.router.NoRoute(gin.WrapH(http.StripPrefix("", http.FileServer(fs))))
}
