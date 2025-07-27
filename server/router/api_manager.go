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

// RouteDefinition defines an API route and associated handler
type RouteDefinition struct {
	Method      string
	Pattern     string
	Description string
	Handler     func(http.ResponseWriter, *http.Request)
}

// ApiManager manages the API routing and client interactions
type ApiManager struct {
	Clients     map[string]client.ClientInterface
	Templates   *template.Template
	ImagePolicy *policy.ImagePolicy
	Routes      []RouteDefinition
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

	// Define routes after creating the manager so handlers can be properly bound
	manager.defineRoutes()

	return manager
}

// defineRoutes initializes the route definitions
func (m *ApiManager) defineRoutes() {
	m.Routes = []RouteDefinition{
		{Method: "GET", Pattern: "/{$}", Description: "Root endpoint", Handler: m.HandleRoot},
		{Method: "GET", Pattern: "/api/v1/{$}", Description: "API root information", Handler: m.HandleApiRoot},
		{Method: "GET", Pattern: "/api/v1/{registry}/{namespace}/{repository}/{$}", Description: "List tags", Handler: m.HandleListTags},
		{Method: "GET", Pattern: "/api/v1/{registry}/{namespace}/{repository}/{tag}/{$}", Description: "Resource info", Handler: m.HandleResourceInfo},
		{Method: "GET", Pattern: "/api/v1/{registry}/{namespace}/{repository}/{tag}/descriptor/{$}", Description: "Descriptor", Handler: m.HandleDescriptor},
		{Method: "GET", Pattern: "/api/v1/{registry}/{namespace}/{repository}/{tag}/manifest/{$}", Description: "Manifest", Handler: m.HandleManifest},
		{Method: "GET", Pattern: "/api/v1/{registry}/{namespace}/{repository}/{tag}/download/{$}", Description: "Download", Handler: m.HandleDownload},
	}
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
	// Register all routes from our routes data structure
	for _, route := range m.Routes {
		pattern := fmt.Sprintf("%s %s", route.Method, route.Pattern)
		log.Printf("Registering route: %s", pattern)
		mux.HandleFunc(pattern, route.Handler)
	}

	// // Add a catch-all handler for any routes that don't match
	// mux.HandleFunc("GET /api/v1/{path...}", func(w http.ResponseWriter, r *http.Request) {
	// 	log.Printf("404 Not Found: %s", r.URL.Path)
	// 	http.Error(w, "Not Found - Invalid route", http.StatusNotFound)
	// })
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
	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	// Create the endpoints_pattern map dynamically from our routes
	endpointsPattern := make(map[string]string)
	for _, route := range m.Routes {
		// Skip the root and API root routes
		if route.Pattern == "/" || route.Pattern == "/api/v1" {
			continue
		}

		// Create a key based on the description and store the full URL with placeholders intact
		key := strings.ToLower(strings.ReplaceAll(route.Description, " ", "_"))
		cleanPattern := cleanPatternString(route.Pattern)
		endpointsPattern[key] = baseURL + cleanPattern
	}

	// Create API root response
	response := map[string]interface{}{
		"api_version":          "v1",
		"description":          "WordPress Plugin Registry ORAS API",
		"endpoints_pattern":    endpointsPattern,
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
	// Get all path values using the request pattern directly
	pathValues := getPathValues(req, req.Pattern)
	registry := pathValues["registry"]
	namespace := pathValues["namespace"]
	repository := pathValues["repository"]

	// Debug logging
	log.Printf("HandleListTags called with registry=%s, namespace=%s, repository=%s", registry, namespace, repository)

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

	// Build base URL for tag resources
	scheme, host := getServerInfo(req)
	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	// Create a template URL for tags with placeholders
	tagUrlTemplate := baseURL + req.URL.Path + "{tag}"

	// Create endpoints map with links to each tag's resource info
	tagEndpoints := make(map[string]string)
	for _, tag := range tags {
		// Create URL for this tag by simply replacing {tag} with the actual tag
		tagURL := strings.ReplaceAll(tagUrlTemplate, "{tag}", tag)
		tagEndpoints[tag] = tagURL
	}

	// Build response
	response := map[string]interface{}{
		"repository": namespacedRepository,
		"registry":   client.GetRegistry(),
		"tags":       tags,
		"endpoints":  tagEndpoints,
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
	// Get all path values using the request pattern directly
	pathValues := getPathValues(req, req.Pattern)
	registry := pathValues["registry"]
	namespace := pathValues["namespace"]
	repository := pathValues["repository"]
	tag := pathValues["tag"]

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
	// scheme and host
	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	// Create endpoints for this resource directly using the pattern structure
	endpoints := make(map[string]string)
	cleanRequestPattern := cleanPatternString(req.Pattern)
	for _, route := range m.Routes {
		// Clean pattern by removing trailing {$} if present
		cleanRoutePattern := cleanPatternString(route.Pattern)

		// log both
		log.Printf("Checking route: %s against request pattern: %s", cleanRoutePattern, cleanRequestPattern)

		if strings.HasPrefix(cleanRoutePattern, cleanRequestPattern) {
			// Create a key based on the description and store the full URL
			key := strings.ToLower(strings.ReplaceAll(route.Description, " ", "_"))
			// interpolate /api/v1/{registry}/{namespace}/{repository}/{tag}
			endpoints[key] = baseURL + interpolatePattern(cleanRoutePattern, pathValues)
		}
	}

	// Create API directory response
	response := map[string]interface{}{
		"registry":  client.GetRegistry(),
		"resource":  fmt.Sprintf("%s/%s:%s", namespace, repository, tag),
		"endpoints": endpoints,
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
	// Get all path values using the request pattern directly
	pathValues := getPathValues(req, req.Pattern)
	registry := pathValues["registry"]
	namespace := pathValues["namespace"]
	repository := pathValues["repository"]
	tag := pathValues["tag"]

	// Debug logging
	log.Printf("HandleDescriptor called with registry=%s, namespace=%s, repository=%s, tag=%s",
		registry, namespace, repository, tag)

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
	// Get all path values using the request pattern directly
	pathValues := getPathValues(req, req.Pattern)
	registry := pathValues["registry"]
	namespace := pathValues["namespace"]
	repository := pathValues["repository"]
	tag := pathValues["tag"]

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
	// Get all path values using the request pattern directly
	pathValues := getPathValues(req, req.Pattern)
	registry := pathValues["registry"]
	namespace := pathValues["namespace"]
	repository := pathValues["repository"]
	tag := pathValues["tag"]

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

// cleanPatternString removes trailing {$} from a pattern string
func cleanPatternString(pattern string) string {
	// Remove trailing /{$}
	clean := strings.TrimSuffix(pattern, "/{$}")
	// Remove trailing {$} without slash
	clean = strings.TrimSuffix(clean, "{$}")
	return clean
}

func interpolatePattern(pattern string, params map[string]string) string {
	result := pattern
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// getPathValues extracts all path variables from a request based on a route pattern
func getPathValues(req *http.Request, pattern string) map[string]string {
	result := make(map[string]string)

	// Extract variable names from the pattern
	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		// Check if this part is a variable (starts with { and ends with })
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			// Extract variable name without braces
			varName := part[1 : len(part)-1]

			// Skip any suffix like {$}
			if varName == "$" {
				continue
			}

			// Get the value from the request
			value := req.PathValue(varName)
			if value != "" {
				result[varName] = value
			}
		}
	}

	return result
}
