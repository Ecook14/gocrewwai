package tools

import (
	"fmt"
	"plugin"
	"sync"
)

// ToolRegistry manages the global set of available tools.
type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

var GlobalRegistry = &ToolRegistry{
	tools: make(map[string]Tool),
}

func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *ToolRegistry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return t, nil
}

func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []Tool
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// LoadPlugin dynamically loads a tool from a compiled .so file.
// The plugin must export a variable named 'Tool' that implements the Tool interface.
func (r *ToolRegistry) LoadPlugin(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	symbol, err := p.Lookup("Tool")
	if err != nil {
		return fmt.Errorf("failed to lookup symbol 'Tool': %w", err)
	}

	tool, ok := symbol.(Tool)
	if !ok {
		return fmt.Errorf("plugin symbol does not implement Tool interface")
	}

	r.Register(tool)
	return nil
}

// CreateTool instantiates a tool by name with the provided configuration map.
func CreateTool(name string, config map[string]interface{}) (Tool, error) {
	// Elite Pattern: Dynamic tool instantiation from external configuration.
	// This mapping ensures that any tool can be requested by Developers/Users via YAML.
	switch name {
	case "GitHubTool":
		token, _ := config["token"].(string)
		return NewGitHubTool(token), nil
	case "SlackTool":
		token, _ := config["token"].(string)
		return NewSlackTool(token), nil
	case "SerperTool":
		apiKey, _ := config["api_key"].(string)
		return NewSerperTool(apiKey), nil
	case "ExaTool":
		apiKey, _ := config["api_key"].(string)
		return NewExaTool(apiKey), nil
	case "WolframAlphaTool":
		appID, _ := config["app_id"].(string)
		return NewWolframAlphaTool(appID), nil
	case "CodeInterpreterTool":
		var opts []CodeInterpreterOption
		if e2bKey, ok := config["e2b_key"].(string); ok && e2bKey != "" {
			opts = append(opts, WithE2B(e2bKey))
		}
		if useDocker, ok := config["use_docker"].(bool); ok && useDocker {
			image, _ := config["docker_image"].(string)
			opts = append(opts, WithDocker(image))
		}
		if mem, ok := config["memory_mb"].(int); ok {
			opts = append(opts, WithLimits(int64(mem), 1024))
		}
		return NewCodeInterpreterTool(opts...), nil
	case "ArxivTool":
		return NewArxivTool(), nil
	case "WikipediaTool":
		return NewWikipediaTool(), nil
	case "BrowserTool":
		return NewBrowserTool(), nil
	case "MySQLTool":
		dsn, _ := config["dsn"].(string)
		return NewMySQLTool(dsn)
	case "MongoDBTool":
		endpoint, _ := config["endpoint"].(string)
		apiKey, _ := config["api_key"].(string)
		dataSource, _ := config["data_source"].(string)
		database, _ := config["database"].(string)
		return NewMongoDBTool(endpoint, apiKey, dataSource, database), nil
	case "ElasticsearchTool":
		baseURL, _ := config["base_url"].(string)
		var opts []func(*ElasticsearchTool)
		if user, ok := config["username"].(string); ok {
			pass, _ := config["password"].(string)
			opts = append(opts, WithESBasicAuth(user, pass))
		}
		if key, ok := config["api_key"].(string); ok {
			opts = append(opts, WithESAPIKey(key))
		}
		return NewElasticsearchTool(baseURL, opts...), nil
	case "S3Tool":
		endpoint, _ := config["endpoint"].(string)
		accessKey, _ := config["access_key"].(string)
		secretKey, _ := config["secret_key"].(string)
		region, _ := config["region"].(string)
		return NewS3Tool(endpoint, accessKey, secretKey, region), nil
	case "HTTPTool":
		var opts []func(*HTTPTool)
		if baseURL, ok := config["base_url"].(string); ok {
			opts = append(opts, WithHTTPBaseURL(baseURL))
		}
		if headers, ok := config["headers"].(map[string]string); ok {
			opts = append(opts, WithHTTPHeaders(headers))
		}
		return NewHTTPTool(opts...), nil
	case "EmailTool":
		host, _ := config["smtp_host"].(string)
		port := 587
		if p, ok := config["smtp_port"].(int); ok {
			port = p
		}
		username, _ := config["username"].(string)
		password, _ := config["password"].(string)
		from, _ := config["from"].(string)
		return NewEmailTool(host, port, username, password, from), nil
	case "ShellTool":
		return NewShellTool(), nil
	case "JSONTool":
		return NewJSONTool(), nil
	case "RegexTool":
		return NewRegexTool(), nil
	default:
		return nil, fmt.Errorf("unsupported tool for dynamic creation: %s", name)
	}
}
