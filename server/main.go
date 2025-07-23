package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Entry point of the program
func Serve(router *http.ServeMux, port string) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}
	log.Println("Listening...")
	server.ListenAndServe() // Run the http server
}

// InitializeRoutes sets up all the HTTP routes for the server
func InitializeRoutes(client ClientInterface) *http.ServeMux {
	router := NewRouter(client)
	mux := http.NewServeMux()

	// Setup all routes
	router.SetupRoutes(mux)

	return mux
}

// ClientInterface defines the methods a client must implement
type ClientInterface interface {
	GetDescriptor(repository string, tagName string) (*v1.Descriptor, error)
	GetManifest(repository string, tagName string) ([]byte, error)
	GetAnnotations(repository string, tagName string) (map[string]string, error)
	GetFirstLayerReader(repository, tagName string) (*client.LayerInfo, error)
	ListTags(repository string) ([]string, error)
}
