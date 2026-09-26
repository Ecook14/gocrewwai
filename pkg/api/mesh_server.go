package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Ecook14/gocrewwai/pkg/api/mesh"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
)

var standaloneMesh = struct {
	sync.Mutex
	srvs []*grpc.Server
}{}

// StopMeshServers gracefully drains all standalone mesh servers.
func StopMeshServers() {
	standaloneMesh.Lock()
	defer standaloneMesh.Unlock()
	for _, s := range standaloneMesh.srvs {
		s.GracefulStop()
	}
	standaloneMesh.srvs = nil
}

// StartMeshServer starts the gRPC mesh server on the given port with optional mTLS.
// When certPEM and keyPEM are provided, the server enforces mTLS (client cert verification).
// When only serverCert/serverKey are provided without clientCA, server-only TLS is used.
// With no certificates, the server mints an ephemeral self-signed cert unless
// MESH_INSECURE=1 explicitly opts into plaintext (secure by default).
func StartMeshServer(port int, agents []core.Agent, store memory.Store, embedder llm.Embedder, serverCertPEM, serverKeyPEM, clientCAPEM []byte) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	var gsrv *grpc.Server
	switch {
	case len(serverCertPEM) > 0 && len(serverKeyPEM) > 0:
		cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
		if err != nil {
			return fmt.Errorf("mesh: failed to load server TLS key pair: %w", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		if len(clientCAPEM) > 0 {
			caPool := x509.NewCertPool()
			if !caPool.AppendCertsFromPEM(clientCAPEM) {
				return fmt.Errorf("mesh: failed to parse client CA certificate")
			}
			tlsCfg.ClientCAs = caPool
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsCfg.ClientAuth = tls.VerifyClientCertIfGiven
		}
		gsrv = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsCfg)))
	case os.Getenv("MESH_INSECURE") == "1":
		slog.Error("mesh: serving PLAINTEXT — explicitly requested via MESH_INSECURE=1",
			slog.Int("port", port))
		gsrv = grpc.NewServer()
	default:
		certPEM, keyPEM, fp, err := ephemeralServerCert()
		if err != nil {
			return err
		}
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return fmt.Errorf("mesh: ephemeral keypair: %w", err)
		}
		slog.Warn("mesh: no operator certificate — TLS with ephemeral self-signed cert",
			slog.Int("port", port), slog.String("fingerprint", fp))
		gsrv = grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		})))
	}

	srv := NewMeshServer()
	srv.agents = make(map[string]core.Agent)
	for _, a := range agents {
		srv.agents[a.GetRole()] = a
	}
	srv.store = store
	srv.embedder = embedder

	mesh.RegisterMeshServiceServer(gsrv, srv)

	standaloneMesh.Lock()
	standaloneMesh.srvs = append(standaloneMesh.srvs, gsrv)
	standaloneMesh.Unlock()

	fmt.Printf("\n%s AGENT MESH gRPC SERVER STARTING ON PORT %d\n", "🕸️", port)
	return gsrv.Serve(lis)
}
