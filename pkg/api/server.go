package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// Server represents the API and Streaming server.
type Server struct {
	router          *gin.Engine
	authToken       string
	sessions        map[string]SessionState
	mu              sync.RWMutex
	checkpointStore crew.CheckpointStore
	logger          *telemetry.AuditLogger
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithCheckpointStore sets the checkpoint backend for session persistence.
func WithCheckpointStore(store crew.CheckpointStore) ServerOption {
	return func(s *Server) { s.checkpointStore = store }
}

// WithLogger sets the audit logger for the server.
func WithLogger(logger *telemetry.AuditLogger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// NewServer creates a new API server with CORS and authentication enabled.
func NewServer(opts ...ServerOption) *Server {
	r := gin.Default()

	// Limit request body size to prevent memory exhaustion from
	// oversized JSON payloads (e.g. huge crew definitions).
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1MB
		c.Next()
	})

	authToken := os.Getenv("API_AUTH_TOKEN")
	if authToken == "" {
		authToken = "gocrewwai-insecure-default-change-me"
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(authenticationMiddleware(authToken))

	s := &Server{
		router:    r,
		authToken: authToken,
		sessions:  make(map[string]SessionState),
		shutdown:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.logger == nil {
		s.logger = telemetry.NewAuditLogger(os.Stdout, telemetry.DefaultAuditLogMaxLen())
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

// Run starts the server on the given address. It blocks until shutdown.
func (s *Server) Run(addr string) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.shutdown
	}()

	// Configure HTTP server with timeouts to prevent slowloris and
	// hanging connections from exhausting resources.
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}

// Shutdown initiates a graceful shutdown of the server.
func (s *Server) Shutdown() {
	close(s.shutdown)
	s.wg.Wait()
}

// authenticationMiddleware returns a gin middleware that validates the
// Authorization header against the server's authToken. Set API_AUTH_TOKEN
// to a non-empty value in production; the default is intentionally insecure.
func authenticationMiddleware(authToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authToken == "" {
			// No token configured — skip auth (insecure, for dev only)
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		expected := []byte("Bearer " + authToken)
		actual := []byte(authHeader)
		if len(actual) == 0 || subtle.ConstantTimeCompare(actual, expected) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing API_AUTH_TOKEN",
			})
			return
		}
		c.Next()
	}
}

// ServeStatic enables the delivery of static files from an embedded filesystem.
// Returns an error if fs is nil (which happens when the web flag is set but
// no embedded UI is available).
func (s *Server) ServeStatic(fs http.FileSystem) error {
    if fs == nil {
        return fmt.Errorf("static file server: nil filesystem — embed UI source into web/embed.go or set --web flag with real assets")
    }
    s.router.NoRoute(gin.WrapH(http.StripPrefix("/", http.FileServer(fs))))
    return nil
}
