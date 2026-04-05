package memory

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	// 1. Dial remote mesh node
	conn, err := grpc.Dial(s.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
