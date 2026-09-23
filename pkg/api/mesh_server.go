package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/api/mesh"
)

// StartMeshServer starts the gRPC mesh server on the given port with optional mTLS.
// When certPEM and keyPEM are provided, the server enforces mTLS (client cert verification).
// When only serverCert/serverKey are provided without clientCA, server-only TLS is used.
// When nil certificates are provided, the server starts in plaintext mode with an audit warning.
func StartMeshServer(port int, agents []core.Agent, store memory.Store, embedder llm.Embedder, serverCertPEM, serverKeyPEM, clientCAPEM []byte) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	var gsrv *grpc.Server
	if len(serverCertPEM) > 0 && len(serverKeyPEM) > 0 {
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
	} else {
		slog.Warn("mesh: starting without TLS — mesh traffic is unencrypted", slog.Int("port", port))
		gsrv = grpc.NewServer()
	}

	srv := NewMeshServer()
	srv.agents = make(map[string]core.Agent)
	for _, a := range agents {
		srv.agents[a.GetRole()] = a
	}
	srv.store = store
	srv.embedder = embedder

	mesh.RegisterMeshServiceServer(gsrv, srv)

	fmt.Printf("\n%s AGENT MESH gRPC SERVER STARTING ON PORT %d\n", "🕸️", port)
	return gsrv.Serve(lis)
}
