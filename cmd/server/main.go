package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/api"
	"github.com/Ecook14/gocrewwai/pkg/config"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

func main() {
	// 0. Parse Command Line Flags
	apiPortFlag := flag.String("api-port", "", "Port for the Gin REST API (default: 8080 or API_PORT env)")
	meshPortFlag := flag.String("mesh-port", "", "Port for the gRPC Agent Mesh (default: 50051 or MESH_PORT env)")
	helpFlag := flag.Bool("help", false, "Show this help message")

	flag.Usage = func() {
		fmt.Printf("Gocrewwai Elite Server\n\n")
		fmt.Printf("Usage:\n")
		fmt.Printf("  server [options]\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nEnvironment Variables:\n")
		fmt.Printf("  API_PORT    Override REST API port\n")
		fmt.Printf("  MESH_PORT   Override gRPC Mesh port\n")
		fmt.Printf("  REDIS_ADDR  Redis address for caching\n")
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// 1. Load System Configuration
	cfg := config.Get()
	if cfg == nil {
		log.Fatalf("❌ Failed to initialize configuration")
	}

	// 1.1 Print Elite Banner
	fmt.Println(`
   ______                      _       __  ___ 
  / ____/________ _      __   | |     / / /   |
 / /   / ___/ _ \ | /| / /   | | /| / / / /| |
/ /___/ /  /  __/ |/ |/ /    | |/ |/ / / ___ |
\____/_/   \___/|__/|__/     |__/|__/ /_/  |_| v0.9 (Elite)
                                               `)
	log.Printf("🛠️  Engine: Gocrewwai | Mode: Multi-Service Orchestrator")
	log.Printf("📂 Config: %s", os.Getenv("CREW_CONFIG_PATH"))
	if os.Getenv("CREW_CONFIG_PATH") == "" {
		log.Printf("📂 Config: (default) config.json")
	}
	fmt.Println("---------------------------------------------------------")

	// 1.5 Initialize Persistence (Breaks Import Cycle)
	if cfg.Persistence.Sessions.Driver != "" {
		if _, err := core.InitSessionManager(cfg.Persistence.Sessions.Driver, cfg.Persistence.Sessions.ConnectionString); err != nil {
			fmt.Printf("⚠️  Persistence initialization failed: %v\n", err)
		} else {
			fmt.Printf("💾 Persistence layer initialized (%s)\n", cfg.Persistence.Sessions.Driver)
		}
	}

	// 1.6 Initialize Telemetry (Breaks Import Cycle)
	if cfg.Observability.Enabled {
		tCfg := telemetry.TelemetryConfig{
			Enabled:           cfg.Observability.Enabled,
			ServiceName:       cfg.Observability.ServiceName,
			Exporter:          cfg.Observability.Tracing.Exporter,
			SamplingRate:      cfg.Observability.Tracing.SamplingRate,
			PrometheusEnabled: cfg.Observability.Prometheus.Enabled,
			PrometheusPort:    cfg.Observability.Prometheus.Port,
		}
		if _, err := telemetry.InitTelemetry(tCfg); err != nil {
			fmt.Printf("⚠️  Telemetry initialization failed: %v\n", err)
		} else {
			fmt.Printf("📊 Telemetry initialized (Service: %s)\n", cfg.Observability.ServiceName)
		}
	}

	// 2. Setup Gin API Server (for Visual Builder and SSE)
	server := api.NewServer()
	
	// 3. Resolve Ports (Flag > Env > Default)
	apiPort := *apiPortFlag
	if apiPort == "" {
		apiPort = os.Getenv("API_PORT")
	}
	if apiPort == "" {
		apiPort = "8080"
	}

	meshPort := *meshPortFlag
	if meshPort == "" {
		meshPort = os.Getenv("MESH_PORT")
	}
	if meshPort == "" {
		meshPort = "50051"
	}

	// 4. Setup gRPC Mesh Server
	meshServer := api.NewMeshServer()

	// 5. Launch Mesh Server in background
	go func() {
		fmt.Printf("🕸  Agent Mesh Server starting on port %s...\n", meshPort)
		if err := meshServer.Start(meshPort); err != nil {
			fmt.Printf("❌ Mesh Server failed: %v\n", err)
		}
	}()

	// 6. Launch Gin API Server (Blocking)
	fmt.Printf("🚀 Crew-GO API Engine starting on port %s...\n", apiPort)
	fmt.Printf("📡 SSE Streaming enabled at /api/v1/stream/:id\n")
	fmt.Println("---------------------------------------------------------")
	
	if err := server.Run(":" + apiPort); err != nil {
		log.Fatalf("❌ API Server failed: %v", err)
	}
}
