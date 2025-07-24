package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server/policy"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server/router"
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
		fmt.Printf("WordPress Plugin Registry ORAS v%s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", Date)
		return
	}

	// Initialize and start the server
	Initialize()
}

// Initialize creates a new client and server based on environment variables
func Initialize() {
	// Get required environment variables
	registry := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY")
	if registry == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY environment variable is not set")
	}

	registryUsername := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME")
	if registryUsername == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME environment variable is not set")
	}

	registryPassword := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD")
	if registryPassword == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD environment variable is not set")
	}

	// Get port with default fallback
	port := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}

	// Load image policy (if file exists)
	var imagePolicy *policy.ImagePolicy
	policyPath := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_POLICY_PATH")
	if policyPath == "" {
		policyPath = "config/image_policy.yaml"
	}

	imagePolicy, err := policy.LoadImagePolicy(policyPath)
	if err != nil {
		log.Printf("Warning: Could not load image policy from %s: %v", policyPath, err)
		log.Println("Running without image policy restrictions")
		imagePolicy = &policy.ImagePolicy{} // Empty policy
	} else {
		log.Printf("Loaded image policy with %d allowed and %d blocked images",
			len(imagePolicy.AllowedImages), len(imagePolicy.BlockedImages))
	}

	// Create client
	apiClient := client.NewClient(
		registry,
		registryUsername,
		registryPassword,
	)

	// Create router with client and image policy
	router := router.NewRouter(apiClient, imagePolicy)

	// Create mux and set up routes using the router
	mux := http.NewServeMux()
	router.SetupRoutes(mux)

	// Start the server with the configured mux
	Serve(mux, port)
}

// Entry point of the program
func Serve(handler http.Handler, port string) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}
	log.Println("Listening...")
	server.ListenAndServe() // Run the http server
}
