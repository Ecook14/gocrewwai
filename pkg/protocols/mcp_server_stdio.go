package protocols

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// StdioServer allows running an MCPServer over standard I/O.
type StdioServer struct {
	Server *MCPServer
}

func NewStdioServer(server *MCPServer) *StdioServer {
	return &StdioServer{Server: server}
}

// Serve reads JSON-RPC requests from stdin and writes responses to stdout.
func (s *StdioServer) Serve(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(os.Stdout)

	slog.Info("🚀 MCP Stdio Server started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var req struct {
				JSONRPC string                 `json:"jsonrpc"`
				ID      interface{}            `json:"id"`
				Method  string                 `json:"method"`
				Params  map[string]interface{} `json:"params"`
			}

			if err := decoder.Decode(&req); err != nil {
				if err == io.EOF {
					return nil
				}
				slog.Error("mcp: stdio decode error", slog.Any("error", err))
				continue
			}

			// Handle the request
			var resp interface{}
			var err error

			switch req.Method {
			case "initialize":
				resp = map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]bool{"listChanged": true},
					},
					"serverInfo": map[string]string{
						"name":    "gocrew-mcp-stdio",
						"version": "1.0.0",
					},
				}
			case "tools/list":
				s.Server.mu.RLock()
				tools := make([]MCPToolDefinition, 0, len(s.Server.tools))
				for _, t := range s.Server.tools {
					tools = append(tools, t.Definition)
				}
				s.Server.mu.RUnlock()
				resp = map[string]interface{}{"tools": tools}
			case "tools/call":
				name, _ := req.Params["name"].(string)
				args, _ := req.Params["arguments"].(map[string]interface{})
				s.Server.mu.RLock()
				tool, ok := s.Server.tools[name]
				s.Server.mu.RUnlock()
				if !ok {
					resp = map[string]interface{}{
						"error": map[string]interface{}{
							"code":    -32602,
							"message": fmt.Sprintf("Tool not found: %s", name),
						},
					}
				} else {
					resp, err = tool.Handler(ctx, args)
					if err != nil {
						resp = &MCPToolResult{
							Content: []MCPContent{{Type: "text", Text: err.Error()}},
							IsError: true,
						}
					}
				}
			default:
				resp = map[string]interface{}{
					"error": map[string]interface{}{
						"code":    -32601,
						"message": fmt.Sprintf("Method not found: %s", req.Method),
					},
				}
			}

			// Send response
			rpcResp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
			}
			if errResp, ok := resp.(map[string]interface{}); ok && errResp["error"] != nil {
				rpcResp["error"] = errResp["error"]
			} else {
				rpcResp["result"] = resp
			}

			if err := encoder.Encode(rpcResp); err != nil {
				slog.Error("mcp: stdio encode error", slog.Any("error", err))
			}
		}
	}
}
