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

// Router handles all HTTP routes and contains the dependencies needed for handlers
type Router struct {
	Client      client.ClientInterface
	Templates   *template.Template
	ImagePolicy *policy.ImagePolicy
}

func NewRouter(client client.ClientInterface, imagePolicy *policy.ImagePolicy) RouterInterface {
	r := &Router{
		Client:      client,
		ImagePolicy: imagePolicy,
	}

	// Load templates
	if err := r.loadTemplates(); err != nil {
		log.Printf("Warning: %v", err)
	}

	return r
}

// loadTemplates loads all templates from the templates directory
func (r *Router) loadTemplates() error {
	var err error
	r.Templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("error loading templates: %v", err)
	}
	return nil
}

// HandleRoot handles the root endpoint
func (r *Router) HandleRoot(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
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
	if r.Templates != nil {
		if err := r.Templates.ExecuteTemplate(w, "index.html", data); err != nil {
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

// HandleAPIRoot handles the API root endpoint
func (r *Router) HandleAPIRoot(w http.ResponseWriter, req *http.Request) {
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
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleListTags handles the list tags endpoint
func (r *Router) HandleListTags(w http.ResponseWriter, req *http.Request) {
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)
	tags, err := r.Client.ListTags(namespacedRepository)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := map[string]interface{}{
		"repository": namespacedRepository,
		"tags":       tags,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleResourceInfo handles the resource info endpoint
func (r *Router) HandleResourceInfo(w http.ResponseWriter, req *http.Request) {
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")

	// Build base URL for this resource
	scheme, host := getServerInfo(req)
	baseURL := fmt.Sprintf("%s://%s/api/v1/%s/%s/%s",
		scheme, host, namespace, repository, tag)

	// Create API directory response
	response := map[string]interface{}{
		"resource": fmt.Sprintf("%s/%s:%s", namespace, repository, tag),
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

// HandleDescriptor handles the descriptor endpoint
func (r *Router) HandleDescriptor(w http.ResponseWriter, req *http.Request) {
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Check image policy
	if !r.checkImagePolicy(w, req, namespace, repository, tag) {
		return
	}

	desc, err := r.Client.GetDescriptor(namespacedRepository, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// log the description
	log.Printf("Description for %s/%s:%s: %v", namespace, repository, tag, desc)
	w.Header().Set("Content-Type", "application/json")
	// Marshal description to JSON
	w.WriteHeader(http.StatusOK) // Set status code to 200 OK
	// Use a JSON encoder to write the description
	if err := json.NewEncoder(w).Encode(desc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleManifest handles the manifest endpoint
func (r *Router) HandleManifest(w http.ResponseWriter, req *http.Request) {
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Check image policy
	if !r.checkImagePolicy(w, req, namespace, repository, tag) {
		return
	}

	content, err := r.Client.GetManifest(namespacedRepository, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Write the content as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Set status code to 200 OK
	// Write the content to the response
	if _, err := w.Write(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleDownload handles the download endpoint
func (r *Router) HandleDownload(w http.ResponseWriter, req *http.Request) {
	namespace := req.PathValue("namespace")
	repository := req.PathValue("repository")
	tag := req.PathValue("tag")
	namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

	// Check image policy
	if !r.checkImagePolicy(w, req, namespace, repository, tag) {
		return
	}

	// Get the layer info which includes all metadata and the reader
	layerInfo, err := r.Client.GetFirstLayerReader(namespacedRepository, tag)
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

	// Set the content type from the layer's media type
	w.Header().Set("Content-Type", layerInfo.GetMediaType())
	// Set Content-Disposition header to make the browser download with the correct filename
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, layerInfo.GetFilename()))
	// Set Content-Length header for better download handling
	w.Header().Set("Content-Length", fmt.Sprintf("%d", layerInfo.GetSize()))

	w.WriteHeader(http.StatusOK) // Set status code to 200 OK
	// Write the content to the response
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

// checkImagePolicy checks if the requested image is allowed by policy
func (r *Router) checkImagePolicy(w http.ResponseWriter, req *http.Request, namespace, repository, tag string) bool {
	// If no policy is configured, allow all images
	if r.ImagePolicy == nil || (len(r.ImagePolicy.AllowedImages) == 0 && len(r.ImagePolicy.BlockedImages) == 0) {
		return true
	}

	// Construct the full image reference for policy checking
	registry := req.Host // Use the host from the request as the registry
	if r.Client.GetRegistry() != "" {
		// If the client has a registry configured, use that instead
		registry = r.Client.GetRegistry()
	}

	fullImageRef := fmt.Sprintf("%s/%s/%s:%s", registry, namespace, repository, tag)

	// Check if the image is allowed by policy
	if !policy.IsAllowed(fullImageRef, r.ImagePolicy) {
		log.Printf("Access denied to image %s by policy", fullImageRef)
		http.Error(w, "Access to this image is denied by policy", http.StatusForbidden)
		return false
	}

	return true
}

// SetupRoutes registers all the HTTP routes for the server
func (r *Router) SetupRoutes(mux *http.ServeMux) {
	// Register all route handlers
	mux.HandleFunc("GET /", r.HandleRoot)
	mux.HandleFunc("GET /api/v1", r.HandleAPIRoot)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/", r.HandleListTags)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}", r.HandleResourceInfo)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/descriptor", r.HandleDescriptor)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/manifest", r.HandleManifest)
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/download", r.HandleDownload)
}
