package protocols

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fakeWebMCPPage = `
<!DOCTYPE html>
<html>
<head>
	<title>Test WebMCP Service</title>
	<script type="application/mcp+json">
	{
		"version": "1.0",
		"capabilities": {},
		"tools": [
			{
				"name": "get_weather",
				"description": "Gets the current weather for a location",
				"endpoint": "/api/weather",
				"method": "GET",
				"inputSchema": {
					"type": "object",
					"properties": {
						"location": {"type": "string"}
					},
					"required": ["location"]
				}
			},
			{
				"name": "book_flight",
				"description": "Books a flight",
				"endpoint": "/api/book",
				"method": "POST",
				"inputSchema": {
					"type": "object",
					"properties": {
						"destination": {"type": "string"}
					}
				}
			}
		]
	}
	</script>
</head>
<body><h1>Hello World</h1></body>
</html>
`

func TestWebMCPDiscoverAndExecute(t *testing.T) {
	// 1. Setup a fake HTTP server that acts as a WebMCP host
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, fakeWebMCPPage)
			return
		}
		if r.URL.Path == "/api/weather" {
			if r.Method != http.MethodGet {
				t.Fatalf("Expected GET, got %s", r.Method)
			}
			location := r.URL.Query().Get("location")
			if location == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"temp": 72, "conditions": "sunny"}`)
			return
		}
		if r.URL.Path == "/api/book" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if body["destination"] != "Paris" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"booking_id": "XYZ123"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewWebMCPClient()
	ctx := context.Background()

	// 2. Test Discovery
	tools, err := client.DiscoverTools(ctx, ts.URL)
	if err != nil {
		t.Fatalf("Failed to discover tools: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "get_weather" {
		t.Errorf("Expected first tool to be get_weather, got %s", tools[0].Name)
	}

	// 3. Test GET Execution
	var weatherTool *WebMCPToolDeclaration
	for _, dt := range tools {
		if dt.Name == "get_weather" {
			weatherTool = &dt
			break
		}
	}

	if weatherTool == nil {
		t.Fatal("Failed to find get_weather in discovered tools")
	}

	// Notice how endpoint normalization appends the ts.URL
	if !strings.HasPrefix(weatherTool.Endpoint, ts.URL) {
		t.Errorf("Expected endpoint to be normalized with base URL, got: %s", weatherTool.Endpoint)
	}

	weatherResp, err := client.ExecuteTool(ctx, *weatherTool, map[string]interface{}{
		"location": "San Francisco",
	})
	if err != nil {
		t.Fatalf("Failed to execute weather tool: %v", err)
	}

	if !strings.Contains(string(weatherResp), "72") {
		t.Errorf("Unexpected weather response: %s", string(weatherResp))
	}

	// 4. Test POST Execution
	var bookTool *WebMCPToolDeclaration
	for _, dt := range tools {
		if dt.Name == "book_flight" {
			bookTool = &dt
			break
		}
	}

	bookResp, err := client.ExecuteTool(ctx, *bookTool, map[string]interface{}{
		"destination": "Paris",
	})
	if err != nil {
		t.Fatalf("Failed to execute book tool: %v", err)
	}

	if !strings.Contains(string(bookResp), "XYZ123") {
		t.Errorf("Unexpected booking response: %s", string(bookResp))
	}
}
