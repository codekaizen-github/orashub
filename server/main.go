package server

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Version information - will be set during build via ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

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

	// Create client
	apiClient := client.NewClient(
		registry,
		registryUsername,
		registryPassword,
	)

	// Create router with client
	router := NewRouter(apiClient)

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

// ClientInterface defines the methods a client must implement
type ClientInterface interface {
	GetDescriptor(repository string, tagName string) (*v1.Descriptor, error)
	GetManifest(repository string, tagName string) ([]byte, error)
	GetFirstLayerReader(repository, tagName string) (*client.LayerInfo, error)
	ListTags(repository string) ([]string, error)
}
