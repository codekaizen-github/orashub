package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server/policy"
)

// Custom error types
var (
	ErrRegistryNotFound  = errors.New("registry not found")
	ErrNoRegistryClients = errors.New("no registry clients available")
)

// ApiManager manages the API routing and client interactions
type ApiManager struct {
	Clients     map[string]client.ClientInterface
	Templates   *template.Template
	ImagePolicy *policy.ImagePolicy
}

// NewApiManager creates a new API manager with the given configuration
func NewApiManager(config *policy.ConfigFile, imagePolicy *policy.ImagePolicy, templates *template.Template) *ApiManager {
	// Check if there are any registries configured - this is a fatal error if not
	if len(config.Registries) == 0 {
		log.Fatalf("Fatal error: No registries configured. Please specify at least one registry in the configuration.")
	}

	manager := &ApiManager{
		Clients:     make(map[string]client.ClientInterface),
		ImagePolicy: imagePolicy,
		Templates:   templates,
	}

	// Create clients for each registry in the config
	for _, registry := range config.Registries {

		// Create client for this registry
		apiClient := client.NewClient(
			registry.Name,
			registry.Username,
			registry.Password,
		)

		// Store client in map
		manager.Clients[registry.Name] = apiClient
	}

	return manager
}

// LoadTemplates loads all templates from the templates directory
// This is a utility function to load templates before creating an ApiManager
func LoadTemplates(templatesPath string) (*template.Template, error) {
	templates, err := template.ParseGlob(filepath.Join(templatesPath, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("error loading templates: %v", err)
	}
	return templates, nil
}

// SetupRoutes registers all HTTP routes for the server
func (m *ApiManager) SetupRoutes(mux *http.ServeMux) {
	// Registry-specific routes - these handle valid API endpoints
	mux.HandleFunc("GET /api/v1/{registry}/{namespace}/{repository}/", m.HandleListTags)
	mux.HandleFunc("GET /api/v1/{registry}/{namespace}/{repository}/{tag}/descriptor", m.HandleDescriptor)
	mux.HandleFunc("GET /api/v1/{registry}/{namespace}/{repository}/{tag}/manifest", m.HandleManifest)
	mux.HandleFunc("GET /api/v1/{registry}/{namespace}/{repository}/{tag}/download", m.HandleDownload)
	mux.HandleFunc("GET /api/v1/{registry}/{namespace}/{repository}/{tag}", m.HandleResourceInfo)

	// Base routes - these are exact matches
	mux.HandleFunc("GET /", m.HandleRoot)
	mux.HandleFunc("GET /api/v1", m.HandleApiRoot)

	// Set custom NotFound handler for unmatched routes
	mux.HandleFunc("GET /api/{rest...}", m.HandleInvalidRoute)
}

// getClient returns the client for the specified registry
// Returns error of type ErrRegistryNotFound if the registry was not found
// Returns error of type ErrNoRegistryClients if no clients are available
func (m *ApiManager) getClient(registry string) (client.ClientInterface, error) {
	// Try to get the client for the specified registry
	if client, ok := m.Clients[registry]; ok {
		return client, nil
	}

	// If the registry doesn't exist in our clients map
	return nil, fmt.Errorf("%w: '%s'", ErrRegistryNotFound, registry)
} // HandleRoot handles the root endpoint
func (m *ApiManager) HandleRoot(w http.ResponseWriter, req *http.Request) {

	// Check if we have any clients configured
	if len(m.Clients) == 0 {
		http.Error(w, "No registry clients configured", http.StatusServiceUnavailable)
		return
	}

	// Get server information for API URL
	scheme, host := getServerInfo(req)
	apiURL := fmt.Sprintf("%s://%s/api/v1", scheme, host)

	// Define template data
	data := struct {
		ApiURL string
	}{
		ApiURL: apiURL,
	}

	// Execute template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if m.Templates != nil {
		if err := m.Templates.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("Error executing template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	} else {
		// No templates were loaded, use fallback
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleApiRoot handles the API root endpoint
func (m *ApiManager) HandleApiRoot(w http.ResponseWriter, req *http.Request) {
	// Check if we have any clients configured
	if len(m.Clients) == 0 {
		http.Error(w, "No registry clients configured", http.StatusServiceUnavailable)
		return
	}

	scheme, host := getServerInfo(req)
	baseURL := fmt.Sprintf("%s://%s/api/v1", scheme, host)

	// Create API root response
	response := map[string]interface{}{
		"api_version": "v1",
		"description": "WordPress Plugin Registry ORAS API",
		"usage": map[string]string{
			"resource_path": baseURL + "{registry}/{namespace}/{repository}/{tag}",
			"example":       baseURL + "ghcr.io/codekaizen-github/wp-github-gist-block/latest",
		},
		"endpoints_pattern": map[string]string{
			"list_tags":     baseURL + "{registry}/{namespace}/{repository}/",
			"resource_info": baseURL + "{registry}/{namespace}/{repository}/{tag}",
			"descriptor":    baseURL + "{registry}/{namespace}/{repository}/{tag}/descriptor",
			"manifest":      baseURL + "{registry}/{namespace}/{repository}/{tag}/manifest",
			"annotations":   baseURL + "{registry}/{namespace}/{repository}/{tag}/annotations",
			"download":      baseURL + "{registry}/{namespace}/{repository}/{tag}/download",
		},
		"available_registries": m.getAvailableRegistries(),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// getAvailableRegistries returns a list of available registry names
func (m *ApiManager) getAvailableRegistries() []string {
	registries := make([]string, 0, len(m.Clients))
	for registry := range m.Clients {
		registries = append(registries, registry)
	}
	return registries
}

// checkImagePolicy checks if the requested repository is allowed by policy
func (m *ApiManager) checkImagePolicy(w http.ResponseWriter, req *http.Request, registry, namespace, repository string) bool {
	// If no policy is configured, allow all repositories
	if m.ImagePolicy == nil || (len(m.ImagePolicy.AllowedRepositories) == 0 && len(m.ImagePolicy.BlockedRepositories) == 0) {
		return true
	}

	// Registry should never be empty - this is a requirement
	if registry == "" {
		log.Printf("Error: Empty registry in checkImagePolicy")
		http.Error(w, "Registry is required for policy check", http.StatusBadRequest)
		return false
	}

	// Create repository path without the tag
	// Important: Do NOT include the registry in the path again if it's already part of namespace
	if strings.HasPrefix(namespace, registry+"/") {
		// The namespace already contains the registry, don't duplicate
		repositoryPath := fmt.Sprintf("%s/%s", namespace, repository)
		log.Printf("Repository path for policy check: %s", repositoryPath)

		// Check if the repository is allowed by policy
		if !policy.IsAllowed(repositoryPath, m.ImagePolicy) {
			log.Printf("Access denied to repository %s by policy", repositoryPath)
			http.Error(w, "Access to this repository is denied by policy", http.StatusForbidden)
			return false
		}
	} else {
		// Normal case, combine registry with namespace and repository
		repositoryPath := fmt.Sprintf("%s/%s/%s", registry, namespace, repository)
		log.Printf("Repository path for policy check: %s", repositoryPath)

		// Check if the repository is allowed by policy
		if !policy.IsAllowed(repositoryPath, m.ImagePolicy) {
			log.Printf("Access denied to repository %s by policy", repositoryPath)
			http.Error(w, "Access to this repository is denied by policy", http.StatusForbidden)
			return false
		}
	}

	return true
}

// HandleListTags handles the list tags endpoint for both default and registry-specific routes
func (m *ApiManager) HandleListTags(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry")
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	// Get client
	client, err := m.getClient(registry)
	if err != nil {
		// Handle specific error types
		switch {
		case errors.Is(err, ErrRegistryNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoRegistryClients):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build namespaced repository
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Get tags
	tags, err := client.ListTags(namespacedRepository)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"repository": namespacedRepository,
		"registry":   client.GetRegistry(),
		"tags":       tags,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleResourceInfo handles the resource info endpoint for both default and registry-specific routes
func (m *ApiManager) HandleResourceInfo(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry")
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")
	// Get client
	client, err := m.getClient(registry)
	if err != nil {
		// Handle specific error types
		switch {
		case errors.Is(err, ErrRegistryNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoRegistryClients):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build base URL for this resource
	scheme, host := getServerInfo(req)
	baseURL := fmt.Sprintf("%s://%s/api/v1/%s/%s/%s/%s",
		scheme, host, registry, namespace, repository, tag)

	// Create API directory response
	response := map[string]interface{}{
		"resource": fmt.Sprintf("%s/%s:%s", namespace, repository, tag),
		"registry": client.GetRegistry(),
		"endpoints": map[string]string{
			"self":        baseURL,
			"descriptor":  baseURL + "/descriptor",
			"manifest":    baseURL + "/manifest",
			"annotations": baseURL + "/annotations",
			"download":    baseURL + "/download",
		},
		"description": "WordPress Plugin Registry ORAS API",
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleDescriptor handles the descriptor endpoint for both default and registry-specific routes
func (m *ApiManager) HandleDescriptor(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry")
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client, err := m.getClient(registry)
	if err != nil {
		// Handle specific error types
		switch {
		case errors.Is(err, ErrRegistryNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoRegistryClients):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build namespaced repository
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Get descriptor
	desc, err := client.GetDescriptor(namespacedRepository, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log the description
	log.Printf("Description for %s/%s:%s: %v", namespace, repository, tag, desc)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(desc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleManifest handles the manifest endpoint for both default and registry-specific routes
func (m *ApiManager) HandleManifest(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry")
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")
	// Get client
	client, err := m.getClient(registry)
	if err != nil {
		// Handle specific error types
		switch {
		case errors.Is(err, ErrRegistryNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoRegistryClients):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build namespaced repository
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Get manifest
	content, err := client.GetManifest(namespacedRepository, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleDownload handles the download endpoint for both default and registry-specific routes
func (m *ApiManager) HandleDownload(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry")
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client, err := m.getClient(registry)
	if err != nil {
		// Handle specific error types
		switch {
		case errors.Is(err, ErrRegistryNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoRegistryClients):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build namespaced repository
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Get layer info
	layerInfo, err := client.GetFirstLayerReader(namespacedRepository, tag)
	if err != nil {
		log.Printf("Error getting first layer reader for %s/%s:%s: %v", namespace, repository, tag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if layerInfo == nil {
		log.Printf("No content found for %s/%s:%s", namespace, repository, tag)
		http.Error(w, "no content found for the first layer", http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", layerInfo.GetMediaType())
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, layerInfo.GetFilename()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", layerInfo.GetSize()))

	// Return content
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, layerInfo); err != nil {
		log.Printf("Error copying content to response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Close the content reader
	if err := layerInfo.Close(); err != nil {
		log.Printf("Error closing content reader: %v", err)
	}
}

// HandleInvalidRoute returns a 404 Not Found for routes that don't match any valid patterns
func (m *ApiManager) HandleInvalidRoute(w http.ResponseWriter, req *http.Request) {
	log.Printf("Invalid route accessed: %s", req.URL.Path)
	http.Error(w, "Not Found - Invalid API route", http.StatusNotFound)
}
