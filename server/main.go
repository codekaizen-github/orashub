package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codekaizen-github/orashub/server/policy"
	"github.com/codekaizen-github/orashub/server/router"
)

// Version information - will be set during build via ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Start initializes and starts the server, handling version flags
func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If version flag is set, print version info and exit
	if *versionFlag {
		fmt.Printf("ORASHub v%s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", Date)
		return
	}

	// Initialize and start the server
	Initialize()
}

// Initialize creates a new client and server based on environment variables
func Initialize() {
	// Get port with default fallback
	port := os.Getenv("ORASHUB_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}

	// Get config file path with default fallback
	configPath := os.Getenv("ORASHUB_CONFIG_PATH")
	if configPath == "" {
		log.Fatalf("ORASHUB_CONFIG_PATH environment variable is not set")
	}

	// Load configuration file
	config, err := policy.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Get image policy from the configuration
	imagePolicy := config.GetImagePolicy()

	// Load templates
	templatesPath := os.Getenv("ORASHUB_TEMPLATES_PATH")
	if templatesPath == "" {
		templatesPath = "templates" // Default templates path
	}
	templates, err := router.LoadTemplates(templatesPath)
	if err != nil {
		log.Printf("Warning: Error loading templates: %v", err)
		// Continue without templates - the server can still function for API calls
	}

	// Create API manager
	manager := router.NewApiManager(config, imagePolicy, templates)

	// Create mux and set up routes using the manager
	mux := http.NewServeMux()
	manager.SetupRoutes(mux)

	// Start the server with the configured mux
	Serve(mux, port)
}

// Entry point of the program
func Serve(handler http.Handler, port string) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}
	log.Printf("Server listening on port %s", port)
	log.Fatal(server.ListenAndServe()) // Run the http server
}
