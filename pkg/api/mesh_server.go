package api

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/api/mesh" // Placeholder for generated code
)

// MeshServer implements the MeshService gRPC server.
type MeshServer struct {
	mesh.UnimplementedMeshServiceServer
	agents   map[string]core.Agent
	store    memory.Store
	embedder llm.Embedder
}

func NewMeshServer() *MeshServer {
	return &MeshServer{
		agents: make(map[string]core.Agent),
	}
}

// Start launches the gRPC server on the given port.
func (s *MeshServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	gsrv := grpc.NewServer()
	mesh.RegisterMeshServiceServer(gsrv, s)

	return gsrv.Serve(lis)
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

	// 2. Search
	items, err := s.store.Search(ctx, vector, int(req.K))
	if err != nil {
		return &mesh.SearchResponse{Success: false, ErrorMessage: fmt.Sprintf("search failed: %v", err)}, nil
	}

	// 3. Format Response
	var snippets []*mesh.KnowledgeSnippet
	for _, item := range items {
		snippets = append(snippets, &mesh.KnowledgeSnippet{
			Content: item.Text,
			Source:  fmt.Sprintf("%v", item.Metadata["source"]),
			Metadata: map[string]string{
				"id": item.ID,
			},
		})
	}

	return &mesh.SearchResponse{
		Success:  true,
		Snippets: snippets,
	}, nil
}

func (s *MeshServer) IndexKnowledge(ctx context.Context, req *mesh.IndexRequest) (*mesh.IndexResponse, error) {
	if s.store == nil || s.embedder == nil {
		return &mesh.IndexResponse{Success: false, ErrorMessage: "RAG not configured on this mesh node"}, nil
	}

	// 1. Generate Embedding
	vector, err := s.embedder.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		return &mesh.IndexResponse{Success: false, ErrorMessage: fmt.Sprintf("embedding failed: %v", err)}, nil
	}

	// 2. Add to Store
	item := &memory.MemoryItem{
		Text:   req.Content,
		Vector: vector,
		Metadata: map[string]interface{}{
			"source": req.Source,
		},
	}
	err = s.store.Add(ctx, item)
	if err != nil {
		return &mesh.IndexResponse{Success: false, ErrorMessage: fmt.Sprintf("index failed: %v", err)}, nil
	}

	return &mesh.IndexResponse{
		Success:    true,
		DocumentId: item.ID,
	}, nil
}

// StartMeshServer starts the gRPC server on the given port.
func StartMeshServer(port int, agents []core.Agent, store memory.Store, embedder llm.Embedder) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	s := grpc.NewServer()
	srv := NewMeshServer()
	srv.agents = make(map[string]core.Agent)
	for _, a := range agents {
		srv.agents[a.GetRole()] = a
	}
	srv.store = store
	srv.embedder = embedder

	mesh.RegisterMeshServiceServer(s, srv)

	fmt.Printf("\n🕸️ AGENT MESH gRPC SERVER STARTING ON PORT %d\n", port)
	return s.Serve(lis)
}
