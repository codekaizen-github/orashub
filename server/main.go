package server

import (
	"encoding/json"
	"fmt"
	"io"
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

func InitializeRoutes(client ClientInterface) *http.ServeMux {
	mux := http.NewServeMux()

	// Root endpoint - provides basic information
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiURL := fmt.Sprintf("%s://%s/api/v1", scheme, r.Host)

		// HTML response with basic info and link to API
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>WordPress Plugin Registry ORAS</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; line-height: 1.6; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #2c3e50; }
        a { color: #3498db; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .api-link { display: inline-block; margin-top: 20px; background: #3498db; color: white; padding: 10px 15px; border-radius: 4px; }
        .api-link:hover { background: #2980b9; text-decoration: none; }
        code { background: #f8f8f8; padding: 2px 5px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>WordPress Plugin Registry ORAS</h1>
    <p>A service for storing and retrieving WordPress plugins using OCI Registry As Storage (ORAS).</p>

    <h2>API Access</h2>
    <p>The API is available at: <code>%s</code></p>
    <a href="%s" class="api-link">Explore the API</a>

    <h2>Documentation</h2>
    <p>For more information, please refer to the <a href="https://github.com/codekaizen-github/wordpress-plugin-registry-oras">GitHub repository</a>.</p>
</body>
</html>`, apiURL, apiURL)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	// API root endpoint
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s/api/v1", scheme, r.Host)

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
	})

	// Index endpoint - shows available endpoints for a resource
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")

		// Build base URL for this resource
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s/api/v1/%s/%s/%s",
			scheme, r.Host, namespace, repository, tag)

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
	})

	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/descriptor", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")
		namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)
		desc, err := client.GetDescriptor(namespacedRepository, tag)
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
	})
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/manifest", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")
		namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)
		content, err := client.GetManifest(namespacedRepository, tag)
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
	})
	// Handle annotations endpoint
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/annotations", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")
		namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)
		annotations, err := client.GetAnnotations(namespacedRepository, tag)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// log the annotations
		log.Printf("Annotations for %s/%s:%s: %v", namespace, repository, tag, annotations)
		w.Header().Set("Content-Type", "application/json")
		// Marshal annotations to JSON
		w.WriteHeader(http.StatusOK) // Set status code to 200 OK
		// Use a JSON encoder to write the annotations
		if err := json.NewEncoder(w).Encode(annotations); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}/download", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")
		namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)

		// Get the layer info which includes all metadata and the reader
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

		// Set the content type from the layer's media type
		w.Header().Set("Content-Type", layerInfo.MediaType)
		// Set Content-Disposition header to make the browser download with the correct filename
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, layerInfo.Filename))
		// Set Content-Length header for better download handling
		w.Header().Set("Content-Length", fmt.Sprintf("%d", layerInfo.Size))

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
	})

	return mux
}

type ClientInterface interface {
	GetDescriptor(repository string, tagName string) (*v1.Descriptor, error)
	GetManifest(repository string, tagName string) ([]byte, error)
	GetAnnotations(repository string, tagName string) (map[string]string, error)
	GetFirstLayerReader(repository, tagName string) (*client.LayerInfo, error)
}
