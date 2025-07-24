package router

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server/policy"
)

// RegistryManager manages multiple registry clients and routes requests to the appropriate client
type RegistryManager struct {
	Clients         map[string]client.ClientInterface
	Routers         map[string]RouterInterface
	ImagePolicy     *policy.ImagePolicy
	DefaultRegistry string
}

// NewRegistryManager creates a new registry manager with the given configuration
func NewRegistryManager(config *policy.ConfigFile, imagePolicy *policy.ImagePolicy) *RegistryManager {
	manager := &RegistryManager{
		Clients:     make(map[string]client.ClientInterface),
		Routers:     make(map[string]RouterInterface),
		ImagePolicy: imagePolicy,
	}

	// Create clients for each registry in the config
	for _, registry := range config.Registries {
		// Skip registries with empty credentials
		if registry.Username == "" && registry.Password == "" {
			log.Printf("Skipping registry %s: no credentials provided", registry.Name)
			continue
		}

		// Create client for this registry
		apiClient := client.NewClient(
			registry.Name,
			registry.Username,
			registry.Password,
		)

		// Store client in map
		manager.Clients[registry.Name] = apiClient

		// Create router for this client
		router := NewRouter(apiClient, imagePolicy)
		manager.Routers[registry.Name] = router

		// Set the first registry as default
		if manager.DefaultRegistry == "" {
			manager.DefaultRegistry = registry.Name
		}
	}

	return manager
}

// SetupRoutes registers all HTTP routes for all registries
func (m *RegistryManager) SetupRoutes(mux *http.ServeMux) {
	// Special case: root routes go to the default registry router
	if defaultRouter, ok := m.Routers[m.DefaultRegistry]; ok {
		defaultRouter.SetupRoutes(mux)
	} else if len(m.Routers) > 0 {
		// If no default registry, use the first one
		for _, router := range m.Routers {
			router.SetupRoutes(mux)
			break
		}
	} else {
		// No routers available
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "No registry clients configured", http.StatusServiceUnavailable)
		})
	}

	// Add a registry-specific API route pattern
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/", m.HandleRegistryListTags)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}", m.HandleRegistryResourceInfo)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/descriptor", m.HandleRegistryDescriptor)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/manifest", m.HandleRegistryManifest)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/download", m.HandleRegistryDownload)
}

// GetRouter returns the router for the specified registry, or nil if not found
func (m *RegistryManager) GetRouter(registry string) RouterInterface {
	if router, ok := m.Routers[registry]; ok {
		return router
	}
	return nil
}

// HandleRegistryListTags routes the list tags request to the appropriate registry router
func (m *RegistryManager) HandleRegistryListTags(w http.ResponseWriter, r *http.Request) {
	registry := r.PathValue("registry")
	if router, ok := m.Routers[registry]; ok {
		// Replace the path to remove the registry prefix
		path := strings.Replace(r.URL.Path, fmt.Sprintf("/api/v1/registry/%s", registry), "/api/v1", 1)
		r.URL.Path = path
		router.(*Router).HandleListTags(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Registry %s not found", registry), http.StatusNotFound)
	}
}

// HandleRegistryResourceInfo routes the resource info request to the appropriate registry router
func (m *RegistryManager) HandleRegistryResourceInfo(w http.ResponseWriter, r *http.Request) {
	registry := r.PathValue("registry")
	if router, ok := m.Routers[registry]; ok {
		// Replace the path to remove the registry prefix
		path := strings.Replace(r.URL.Path, fmt.Sprintf("/api/v1/registry/%s", registry), "/api/v1", 1)
		r.URL.Path = path
		router.(*Router).HandleResourceInfo(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Registry %s not found", registry), http.StatusNotFound)
	}
}

// HandleRegistryDescriptor routes the descriptor request to the appropriate registry router
func (m *RegistryManager) HandleRegistryDescriptor(w http.ResponseWriter, r *http.Request) {
	registry := r.PathValue("registry")
	if router, ok := m.Routers[registry]; ok {
		// Replace the path to remove the registry prefix
		path := strings.Replace(r.URL.Path, fmt.Sprintf("/api/v1/registry/%s", registry), "/api/v1", 1)
		r.URL.Path = path
		router.(*Router).HandleDescriptor(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Registry %s not found", registry), http.StatusNotFound)
	}
}

// HandleRegistryManifest routes the manifest request to the appropriate registry router
func (m *RegistryManager) HandleRegistryManifest(w http.ResponseWriter, r *http.Request) {
	registry := r.PathValue("registry")
	if router, ok := m.Routers[registry]; ok {
		// Replace the path to remove the registry prefix
		path := strings.Replace(r.URL.Path, fmt.Sprintf("/api/v1/registry/%s", registry), "/api/v1", 1)
		r.URL.Path = path
		router.(*Router).HandleManifest(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Registry %s not found", registry), http.StatusNotFound)
	}
}

// HandleRegistryDownload routes the download request to the appropriate registry router
func (m *RegistryManager) HandleRegistryDownload(w http.ResponseWriter, r *http.Request) {
	registry := r.PathValue("registry")
	if router, ok := m.Routers[registry]; ok {
		// Replace the path to remove the registry prefix
		path := strings.Replace(r.URL.Path, fmt.Sprintf("/api/v1/registry/%s", registry), "/api/v1", 1)
		r.URL.Path = path
		router.(*Router).HandleDownload(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Registry %s not found", registry), http.StatusNotFound)
	}
}
