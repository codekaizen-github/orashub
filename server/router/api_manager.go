package router

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server/policy"
)

// ApiManager manages the API routing and client interactions
type ApiManager struct {
	Clients         map[string]client.ClientInterface
	Templates       *template.Template
	ImagePolicy     *policy.ImagePolicy
	DefaultRegistry string
}

// NewApiManager creates a new API manager with the given configuration
func NewApiManager(config *policy.ConfigFile, imagePolicy *policy.ImagePolicy) *ApiManager {
	manager := &ApiManager{
		Clients:     make(map[string]client.ClientInterface),
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

		// Set the first registry as default
		if manager.DefaultRegistry == "" {
			manager.DefaultRegistry = registry.Name
		}
	}

	// Load templates
	if err := manager.loadTemplates(); err != nil {
		log.Printf("Warning: %v", err)
	}

	return manager
}

// loadTemplates loads all templates from the templates directory
func (m *ApiManager) loadTemplates() error {
	var err error
	m.Templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("error loading templates: %v", err)
	}
	return nil
}

// SetupRoutes registers all HTTP routes for the server
func (m *ApiManager) SetupRoutes(mux *http.ServeMux) {
	// Base routes
	mux.HandleFunc("GET /", m.HandleRoot)
	mux.HandleFunc("GET /api/v1", m.HandleApiRoot)

	// Default registry routes (no registry prefix)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/", m.HandleListTags)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}", m.HandleResourceInfo)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/descriptor", m.HandleDescriptor)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/manifest", m.HandleManifest)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/download", m.HandleDownload)

	// Registry-specific routes
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/", m.HandleListTags)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}", m.HandleResourceInfo)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/descriptor", m.HandleDescriptor)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/manifest", m.HandleManifest)
	mux.HandleFunc("GET /api/v1/registry/{registry}/{namespace}/{repository}/{tag}/download", m.HandleDownload)
}

// getClient returns the client for the specified registry, or the default client if none specified
func (m *ApiManager) getClient(registry string) client.ClientInterface {
	if registry == "" {
		registry = m.DefaultRegistry
	}

	if client, ok := m.Clients[registry]; ok {
		return client
	}

	// If no client found and we have at least one client, return the first one
	if len(m.Clients) > 0 {
		for _, client := range m.Clients {
			return client
		}
	}

	return nil
}

// HandleRoot handles the root endpoint
func (m *ApiManager) HandleRoot(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

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

	// Execute template from cache
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if m.Templates != nil {
		if err := m.Templates.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("Error executing template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to parsing template directly if cache failed
		tmplPath := filepath.Join("templates", "index.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error executing template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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
			"resource_path": baseURL + "/{namespace}/{repository}/{tag}",
			"example":       baseURL + "/codekaizen-github/wp-github-gist-block/latest",
		},
		"endpoints_pattern": map[string]string{
			"resource_info": baseURL + "/{namespace}/{repository}/{tag}",
			"descriptor":    baseURL + "/{namespace}/{repository}/{tag}/descriptor",
			"manifest":      baseURL + "/{namespace}/{repository}/{tag}/manifest",
			"annotations":   baseURL + "/{namespace}/{repository}/{tag}/annotations",
			"download":      baseURL + "/{namespace}/{repository}/{tag}/download",
		},
		"registry_specific": map[string]string{
			"pattern": baseURL + "/registry/{registry}/{namespace}/{repository}/{tag}",
			"example": baseURL + "/registry/docker.io/library/nginx/latest",
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

	// Construct the full repository reference for policy checking
	registryHost := registry
	if registryHost == "" {
		registryHost = m.DefaultRegistry
	}

	// Create repository path without the tag
	repositoryPath := fmt.Sprintf("%s/%s/%s", registryHost, namespace, repository)

	// Check if the repository is allowed by policy
	if !policy.IsAllowed(repositoryPath, m.ImagePolicy) {
		log.Printf("Access denied to repository %s by policy", repositoryPath)
		http.Error(w, "Access to this repository is denied by policy", http.StatusForbidden)
		return false
	}

	return true
}

// HandleListTags handles the list tags endpoint for both default and registry-specific routes
func (m *ApiManager) HandleListTags(w http.ResponseWriter, req *http.Request) {
	// Extract parameters
	registry := req.PathValue("registry") // Will be empty for default routes
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")

	// Get client
	client := m.getClient(registry)
	if client == nil {
		http.Error(w, "No registry client available", http.StatusServiceUnavailable)
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
	registry := req.PathValue("registry") // Will be empty for default routes
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client := m.getClient(registry)
	if client == nil {
		http.Error(w, "No registry client available", http.StatusServiceUnavailable)
		return
	}

	// Check policy
	if !m.checkImagePolicy(w, req, registry, namespace, repository) {
		return
	}

	// Build base URL for this resource
	scheme, host := getServerInfo(req)
	var baseURL string
	if registry == "" {
		baseURL = fmt.Sprintf("%s://%s/api/v1/%s/%s/%s",
			scheme, host, namespace, repository, tag)
	} else {
		baseURL = fmt.Sprintf("%s://%s/api/v1/registry/%s/%s/%s/%s",
			scheme, host, registry, namespace, repository, tag)
	}

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
	registry := req.PathValue("registry") // Will be empty for default routes
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client := m.getClient(registry)
	if client == nil {
		http.Error(w, "No registry client available", http.StatusServiceUnavailable)
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
	registry := req.PathValue("registry") // Will be empty for default routes
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client := m.getClient(registry)
	if client == nil {
		http.Error(w, "No registry client available", http.StatusServiceUnavailable)
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
	registry := req.PathValue("registry") // Will be empty for default routes
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Get client
	client := m.getClient(registry)
	if client == nil {
		http.Error(w, "No registry client available", http.StatusServiceUnavailable)
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
