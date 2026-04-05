package mesh

import (
	"context"
	"google.golang.org/grpc"
)

// This is a manually maintained STUB to avoid 'not in std' errors 
// until the user runs 'protoc' to generate the official Go code.

type TaskRequest struct {
	TaskDescription string
	AgentRole       string
	AgentGoal       string
	SessionId       string
}

type TaskResponse struct {
	Success      bool
	Output       string
	ErrorMessage string
}

type SearchRequest struct {
	Query      string
	K          int32
	Collection string
}

type KnowledgeSnippet struct {
	Content  string
	Source   string
	Metadata map[string]string
}

type SearchResponse struct {
	Success      bool
	ErrorMessage string
	Snippets     []*KnowledgeSnippet
}

type IndexRequest struct {
	Content string
	Source  string
}

type IndexResponse struct {
	Success    bool
	ErrorMessage string
	DocumentId string
}

type UnimplementedMeshServiceServer struct{}

func (UnimplementedMeshServiceServer) DelegateTask(context.Context, *TaskRequest) (*TaskResponse, error) {
	return nil, nil
}
func (UnimplementedMeshServiceServer) SearchKnowledge(context.Context, *SearchRequest) (*SearchResponse, error) {
	return nil, nil
}
func (UnimplementedMeshServiceServer) IndexKnowledge(context.Context, *IndexRequest) (*IndexResponse, error) {
	return nil, nil
}

type MeshServiceClient interface {
	DelegateTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*TaskResponse, error)
	SearchKnowledge(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
	IndexKnowledge(ctx context.Context, in *IndexRequest, opts ...grpc.CallOption) (*IndexResponse, error)
}

type meshServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMeshServiceClient(cc grpc.ClientConnInterface) MeshServiceClient {
	return &meshServiceClient{cc}
}

func (c *meshServiceClient) DelegateTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*TaskResponse, error) {
	out := new(TaskResponse)
	err := c.cc.Invoke(ctx, "/MeshService/DelegateTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *meshServiceClient) SearchKnowledge(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, "/MeshService/SearchKnowledge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *meshServiceClient) IndexKnowledge(ctx context.Context, in *IndexRequest, opts ...grpc.CallOption) (*IndexResponse, error) {
	out := new(IndexResponse)
	err := c.cc.Invoke(ctx, "/MeshService/IndexKnowledge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type MeshServiceServer interface {
	DelegateTask(context.Context, *TaskRequest) (*TaskResponse, error)
	SearchKnowledge(context.Context, *SearchRequest) (*SearchResponse, error)
	IndexKnowledge(context.Context, *IndexRequest) (*IndexResponse, error)
}

func RegisterMeshServiceServer(s grpc.ServiceRegistrar, srv MeshServiceServer) {
	s.RegisterService(&MeshService_ServiceDesc, srv)
}

var MeshService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "MeshService",
	HandlerType: (*MeshServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DelegateTask",
			Handler:    nil, // Placeholder
		},
	},
}
