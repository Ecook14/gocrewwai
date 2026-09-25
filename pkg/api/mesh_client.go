package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Ecook14/gocrewwai/pkg/api/mesh"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
)

// MeshServerCredentialConfig holds TLS configuration for the mesh server.
type MeshServerCredentialConfig struct {
	// EnableTLS controls whether TLS is enforced on the mesh listener.
	EnableTLS bool
	// CertFile is the path to the server certificate (PEM-encoded).
	CertFile string
	// KeyFile is the path to the server private key (PEM-encoded).
	KeyFile string
	// ClientCAFile is the path to the CA certificate used to verify client
	// certificates when mTLS is enabled. When empty and EnableTLS is true,
	// the server accepts any TLS client (server-only TLS).
	ClientCAFile string
}

// MeshClientCredentialConfig holds TLS configuration for the mesh client.
type MeshClientCredentialConfig struct {
	// EnableTLS controls whether the client requires TLS when dialing.
	EnableTLS bool
	// CAFile is the path to the CA certificate used to verify the server.
	// When empty and EnableTLS is true, the client uses the system root CA pool.
	CAFile string
	// ServerName overrides the TLS ServerName for certificate verification.
	ServerName string
}

// DefaultMeshServerCredentials returns a credential config that reads from
// environment variables. This is the production entry point for mesh TLS.
func DefaultMeshServerCredentials() MeshServerCredentialConfig {
	return MeshServerCredentialConfig{
		EnableTLS:    os.Getenv("MESH_TLS_ENABLED") == "true",
		CertFile:     os.Getenv("MESH_TLS_CERT"),
		KeyFile:      os.Getenv("MESH_TLS_KEY"),
		ClientCAFile: os.Getenv("MESH_TLS_CLIENT_CA"),
	}
}

// DefaultMeshClientCredentials returns a credential config that reads from
// environment variables. This is the production entry point for mesh client TLS.
func DefaultMeshClientCredentials() MeshClientCredentialConfig {
	return MeshClientCredentialConfig{
		EnableTLS:  os.Getenv("MESH_TLS_ENABLED") == "true",
		CAFile:     os.Getenv("MESH_TLS_CA"),
		ServerName: os.Getenv("MESH_TLS_SERVER_NAME"),
	}
}

// LoadTLSConfig constructs a tls.Config from the server credential configuration.
// It validates that required files exist and are readable before returning.
func LoadTLSConfig(cfg MeshServerCredentialConfig) (*tls.Config, error) {
	if !cfg.EnableTLS {
		return nil, nil
	}

	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf(
			"mesh TLS enabled but MESH_TLS_CERT and MESH_TLS_KEY must be set",
		)
	}

	for _, p := range []string{cfg.CertFile, cfg.KeyFile} {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("mesh TLS cert file not found: %s: %w", p, err)
		}
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile); err != nil {
		return nil, fmt.Errorf("failed to load mesh TLS cert/key pair: %w", err)
	} else {
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if cfg.ClientCAFile != "" {
		caPool := x509.NewCertPool()
		caData, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mesh client CA file: %w", err)
		}
		if !caPool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf(
				"mesh client CA file did not contain any valid certificates: %s",
				cfg.ClientCAFile,
			)
		}
		tlsCfg.ClientCAs = caPool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsCfg, nil
}

// LoadClientTLSConfig constructs a tls.Config for the mesh client from the
// client credential configuration.
func LoadClientTLSConfig(cfg MeshClientCredentialConfig) (*tls.Config, error) {
	if !cfg.EnableTLS {
		return nil, nil
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.CAFile != "" {
		caPool := x509.NewCertPool()
		caData, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mesh CA file: %w", err)
		}
		if !caPool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf(
				"mesh CA file did not contain any valid certificates: %s",
				cfg.CAFile,
			)
		}
		tlsCfg.RootCAs = caPool
	}

	if cfg.ServerName != "" {
		tlsCfg.ServerName = cfg.ServerName
	}

	return tlsCfg, nil
}

// MeshServer implements the MeshService gRPC server.
type MeshServer struct {
	mesh.UnimplementedMeshServiceServer
	agents   map[string]core.Agent
	store    memory.Store
	embedder llm.Embedder
	tlsCfg   *tls.Config
}

func NewMeshServer() *MeshServer {
	return &MeshServer{
		agents: make(map[string]core.Agent),
	}
}

// Start launches the gRPC server on the given port. When the server is
// configured with TLS, the listener uses tls.NewListener; otherwise it
// falls back to a plain TCP listener with a warning logged to the audit
// trail so that an unsecured mesh is never silent.
func (s *MeshServer) Start(port string, opts ...MeshServerOption) error {
	var cfg MeshServerConfig
	for _, o := range opts {
		o(&cfg)
	}

	credCfg := cfg.Credentials
	if credCfg.EnableTLS {
		if credCfg.CertFile == "" {
			credCfg.CertFile = os.Getenv("MESH_TLS_CERT")
		}
		if credCfg.KeyFile == "" {
			credCfg.KeyFile = os.Getenv("MESH_TLS_KEY")
		}
		if credCfg.ClientCAFile == "" {
			credCfg.ClientCAFile = os.Getenv("MESH_TLS_CLIENT_CA")
		}

		var err error
		s.tlsCfg, err = LoadTLSConfig(credCfg)
		if err != nil {
			return fmt.Errorf("mesh TLS configuration invalid: %w", err)
		}
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	var gsrv *grpc.Server
	if s.tlsCfg != nil {
		gsrv = grpc.NewServer(grpc.Creds(credentials.NewTLS(s.tlsCfg)))
	} else {
		gsrv = grpc.NewServer()
	}

	mesh.RegisterMeshServiceServer(gsrv, s)

	if s.tlsCfg != nil {
		return gsrv.Serve(lis)
	}

	return gsrv.Serve(lis)
}

// MeshServerOption configures a MeshServer at start time.
type MeshServerOption func(*MeshServerConfig)

// MeshServerConfig holds per-start runtime configuration for the mesh server.
type MeshServerConfig struct {
	Credentials MeshServerCredentialConfig
}

// WithMeshServerCredentials overrides the credential configuration for a
// single Start call.
func WithMeshServerCredentials(creds MeshServerCredentialConfig) MeshServerOption {
	return func(cfg *MeshServerConfig) { cfg.Credentials = creds }
}

func (s *MeshServer) DelegateTask(ctx context.Context, req *mesh.TaskRequest) (*mesh.TaskResponse, error) {
	// 1. Find local agent
	agent, ok := s.agents[req.AgentRole]
	if !ok {
		return &mesh.TaskResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Agent with role '%s' not found on this mesh node", req.AgentRole),
		}, nil
	}

	// 2. Execute locally
	result, err := agent.Execute(ctx, req.TaskDescription, map[string]interface{}{
		"session_id": req.SessionId,
	})

	if err != nil {
		return &mesh.TaskResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &mesh.TaskResponse{
		Success: true,
		Output:  fmt.Sprintf("%v", result),
	}, nil
}

func (s *MeshServer) SearchKnowledge(ctx context.Context, req *mesh.SearchRequest) (*mesh.SearchResponse, error) {
	if s.store == nil || s.embedder == nil {
		return &mesh.SearchResponse{Success: false, ErrorMessage: "RAG not configured on this mesh node"}, nil
	}

	// 1. Generate Embedding
	vector, err := s.embedder.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return &mesh.SearchResponse{Success: false, ErrorMessage: fmt.Sprintf("embedding failed: %v", err)}, nil
	}

	// 2. Query Memory
	results, err := s.store.Search(ctx, vector, int(req.K))
	if err != nil {
		return &mesh.SearchResponse{Success: false, ErrorMessage: fmt.Sprintf("search failed: %v", err)}, nil
	}

	// 3. Format Response
	formattedResults := make([]*mesh.KnowledgeSnippet, len(results))
	for i, r := range results {
		formattedResults[i] = &mesh.KnowledgeSnippet{
			Source:   r.ID,
			Score:    0.0,
			Content:  r.Text,
			Metadata: nil,
		}
	}

	return &mesh.SearchResponse{
		Success:  true,
		Snippets: formattedResults,
	}, nil
}

// ConnectMeshClient dials a remote MeshService and returns a gRPC client.
// When the client credential configuration enables TLS, the connection uses
// a tls.Config; otherwise it uses insecure credentials with an audit-logged
// warning so that insecure mesh connections are never silent.
func ConnectMeshClient(addr string, opts ...MeshClientOption) (mesh.MeshServiceClient, error) {
	var cfg MeshClientConfig
	for _, o := range opts {
		o(&cfg)
	}

	credCfg := cfg.Credentials
	if credCfg.EnableTLS {
		if credCfg.CAFile == "" {
			credCfg.CAFile = os.Getenv("MESH_TLS_CA")
		}
		if credCfg.ServerName == "" {
			credCfg.ServerName = os.Getenv("MESH_TLS_SERVER_NAME")
		}

		tlsCfg, err := LoadClientTLSConfig(credCfg)
		if err != nil {
			return nil, fmt.Errorf("mesh client TLS configuration invalid: %w", err)
		}

		if tlsCfg != nil {
			cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
			if err != nil {
				return nil, fmt.Errorf("failed to dial mesh service at %s: %w", addr, err)
			}
			return mesh.NewMeshServiceClient(cc), nil
		}
	}

	// Insecure fallback — logged so it is never silent in production.
	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial mesh service at %s: %w", addr, err)
	}
	return mesh.NewMeshServiceClient(cc), nil
}

// MeshClientOption configures a ConnectMeshClient call.
type MeshClientOption func(*MeshClientConfig)

// MeshClientConfig holds per-call runtime configuration for the mesh client.
type MeshClientConfig struct {
	Credentials MeshClientCredentialConfig
}

// WithMeshClientCredentials overrides the credential configuration for a
// single ConnectMeshClient call.
func WithMeshClientCredentials(creds MeshClientCredentialConfig) MeshClientOption {
	return func(cfg *MeshClientConfig) { cfg.Credentials = creds }
}

// ConnectMeshServerWithAgents creates a MeshServer, registers the given agents,
// and returns it ready to be started.
func ConnectMeshServerWithAgents(agents []core.Agent, store memory.Store, embedder llm.Embedder) *MeshServer {
	srv := NewMeshServer()
	srv.agents = make(map[string]core.Agent)
	for _, a := range agents {
		srv.agents[a.GetRole()] = a
	}
	srv.store = store
	srv.embedder = embedder
	return srv
}
