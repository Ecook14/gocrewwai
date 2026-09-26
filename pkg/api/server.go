package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// rateLimiter is a minimal per-IP token bucket to protect API routes
// without adding new dependencies. Not as precise as x/time/rate but
// prevents single-client hammering of kickoff/session/SSE endpoints.
type rateLimiter struct {
	mu       sync.Mutex
	hits     map[string][]time.Time
	limit    int
	window   time.Duration
	disabled bool
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: make(map[string][]time.Time), limit: limit, window: window}
}

func (rl *rateLimiter) allow(ip string) bool {
	if rl.disabled || rl.limit <= 0 {
		return true
	}
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	bucket := rl.hits[ip]
	fresh := bucket[:0]
	for _, t := range bucket {
		if now.Sub(t) < rl.window {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= rl.limit {
		rl.hits[ip] = fresh
		return false
	}
	rl.hits[ip] = append(fresh, now)
	return true
}

func (rl *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Health checks are excluded from rate limiting.
		if c.FullPath() == "/api/v1/health" || c.Request.URL.Path == "/api/v1/health" {
			c.Next()
			return
		}
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// Server represents the API and Streaming server.
type Server struct {
	router          *gin.Engine
	authTokens      []string
	sessions        map[string]SessionState
	mu              sync.RWMutex
	checkpointStore crew.CheckpointStore
	logger          *telemetry.AuditLogger
	shutdown        chan struct{}
	wg              sync.WaitGroup
	// Idempotency-Key ledger (in-memory; a restart clears it).
	idemMu        sync.Mutex
	idemKeys      map[string]*idemEntry
	idemBySession map[string]string
}

// idemEntry tracks one Idempotency-Key from acceptance to completion.
type idemEntry struct {
	sessionID string
	owner     string
	done      bool
	status    string
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

	authTokens := resolveAuthTokens()
	if len(authTokens) == 1 && authTokens[0] == "gocrewwai-insecure-default-change-me" {
		// Insecure default retained for dev; documented as requiring change.
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Per-IP rate limiting (100 req/min default) before auth.
	r.Use(newRateLimiter(100, time.Minute).middleware())

	r.Use(authenticationMiddleware(authTokens))

	s := &Server{
		router:        r,
		authTokens:    authTokens,
		sessions:      make(map[string]SessionState),
		shutdown:      make(chan struct{}),
		idemKeys:      make(map[string]*idemEntry),
		idemBySession: make(map[string]string),
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
		"version": "0.9.0",
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

// resolveAuthTokens builds the accepted API token set. API_AUTH_TOKENS
// (comma-separated) supports multiple tenants; API_AUTH_TOKEN is the
// single-token fallback. Sessions are owner-scoped to the presenting token's
// fingerprint, so token A's sessions are invisible to token B (DCR-04).
func resolveAuthTokens() []string {
	if list := os.Getenv("API_AUTH_TOKENS"); list != "" {
		var out []string
		for _, t := range strings.Split(list, ",") {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if t := os.Getenv("API_AUTH_TOKEN"); t != "" {
		return []string{t}
	}
	return []string{"gocrewwai-insecure-default-change-me"}
}

// tokenFingerprint derives a stable, non-reversible owner ID for a token.
func tokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// authenticationMiddleware returns a gin middleware that validates the
// Authorization header against the server's token set using constant-time
// comparison. Set API_AUTH_TOKEN (or API_AUTH_TOKENS for multi-tenant) to a
// non-empty value in production; the default is intentionally insecure.
func authenticationMiddleware(authTokens []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(authTokens) == 0 {
			// No token configured — skip auth (insecure, for dev only)
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		for _, tok := range authTokens {
			expected := []byte("Bearer " + tok)
			actual := []byte(authHeader)
			if len(actual) != 0 && subtle.ConstantTimeCompare(actual, expected) == 1 {
				c.Set("token_fp", tokenFingerprint(tok))
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized: invalid or missing API_AUTH_TOKEN",
		})
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
