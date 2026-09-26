package memory

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"

	"github.com/Ecook14/gocrewwai/pkg/api/mesh" // Placeholder for generated code
)

// RemoteKnowledgeSource allows a local agent to query a remote RAG node over gRPC.
type RemoteKnowledgeSource struct {
	Address    string
	Collection string
	K          int32
}

func NewRemoteKnowledgeSource(address, collection string, k int) *RemoteKnowledgeSource {
	return &RemoteKnowledgeSource{
		Address:    address,
		Collection: collection,
		K:          int32(k),
	}
}

func (s *RemoteKnowledgeSource) Query(ctx context.Context, query string) (string, error) {
	// 1. Dial remote mesh node (TLS when MESH_TLS_CA is set, insecure otherwise)
	conn, err := dialMeshNode(s.Address)
	if err != nil {
		return "", fmt.Errorf("failed to connect to remote RAG node at %s: %w", s.Address, err)
	}
	defer conn.Close()

	// 2. Create client
	client := mesh.NewMeshServiceClient(conn)

	// 3. Search Request
	req := &mesh.SearchRequest{
		Query:      query,
		K:          s.K,
		Collection: s.Collection,
	}

	// 4. Remote Call
	resp, err := client.SearchKnowledge(ctx, req)
	if err != nil {
		return "", fmt.Errorf("remote RAG query failed: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("remote RAG error: %s", resp.ErrorMessage)
	}

	// 5. Format snippet results into a single context string
	var builder strings.Builder
	builder.WriteString("Results from Remote Knowledge Source:\n")
	for i, snippet := range resp.Snippets {
		builder.WriteString(fmt.Sprintf("[%d] Source: %s\nContent: %s\n\n", i+1, snippet.Source, snippet.Content))
	}

	return builder.String(), nil
}

// dialMeshNode mirrors the mesh client credential policy without importing
// pkg/api (which would create an import cycle: api imports memory).
func dialMeshNode(address string) (*grpc.ClientConn, error) {
	return mesh.DialNode(address)
}
