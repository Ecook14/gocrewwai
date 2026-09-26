package mesh

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Mesh transport policy (secure by default):
//
//   - MESH_TLS_CA set → verified TLS against that CA (server name from
//     MESH_TLS_SERVER_NAME, may be empty).
//   - MESH_INSECURE=1 → plaintext with an audit warning (explicit opt-out).
//   - Loopback destination, neither set → TLS with system roots skipped.
//     Same-host dev traffic stays encrypted without a CA bootstrap; this is
//     NOT valid for remote peers.
//   - Remote destination, neither set → error. The caller must provide a CA
//     or explicitly opt out. Failing closed beats silent plaintext (DCR-03).
func DialNode(address string) (*grpc.ClientConn, error) {
	useTLS, skipVerify, err := ResolveTransport(address)
	if err != nil {
		return nil, err
	}
	if !useTLS {
		slog.Warn("mesh: plaintext dial explicitly requested",
			slog.String("address", address))
		return grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if skipVerify {
		slog.Warn("mesh: loopback TLS without verification (dev only)",
			slog.String("address", address))
		return grpc.NewClient(address, grpc.WithTransportCredentials(credentials.NewTLS(
			&tls.Config{InsecureSkipVerify: true}, //nolint:gosec // loopback dev only, see policy above
		)))
	}
	creds, err := credentials.NewClientTLSFromFile(
		os.Getenv("MESH_TLS_CA"), os.Getenv("MESH_TLS_SERVER_NAME"))
	if err != nil {
		return nil, fmt.Errorf("mesh TLS credentials invalid: %w", err)
	}
	return grpc.NewClient(address, grpc.WithTransportCredentials(creds))
}

// ResolveTransport maps an address to (useTLS, skipVerify, err) per the
// policy above. Pure function of address + env, unit tested.
func ResolveTransport(address string) (bool, bool, error) {
	if os.Getenv("MESH_TLS_CA") != "" {
		return true, false, nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address // bare host without port
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "127.") {
		return true, true, nil
	}
	if os.Getenv("MESH_INSECURE") == "1" {
		return false, false, nil
	}
	return false, false, fmt.Errorf(
		"mesh: refusing plaintext dial to remote %q: set MESH_TLS_CA or MESH_INSECURE=1", address)
}
