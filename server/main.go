package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

// Template cache to store parsed templates
var templates *template.Template

// loadTemplates loads all templates from the templates directory
func loadTemplates() error {
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("error loading templates: %v", err)
	}
	return nil
}

// getServerInfo returns the scheme, host, and port to use for API URLs
// It checks environment variables first, then falls back to request values
func getServerInfo(r *http.Request) (scheme, host string) {
	// Check for scheme override from environment variable
	scheme = os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_SCHEME")
	if scheme == "" {
		// Fall back to request scheme
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	// Check for host override from environment variable
	envHost := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_HOST")
	envPort := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_PORT")

	if envHost != "" {
		host = envHost
		// If port is also specified, append it to the host
		if envPort != "" && envPort != "80" && envPort != "443" {
			host = fmt.Sprintf("%s:%s", host, envPort)
		}
	} else {
		// Use the host from the request
		host = r.Host
	}

	return scheme, host
}

func InitializeRoutes(client ClientInterface) *http.ServeMux {
	// Load templates
	if err := loadTemplates(); err != nil {
		log.Printf("Warning: %v", err)
	}

	mux := http.NewServeMux()

	// List tags endpoint
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		namespacedRepository := fmt.Sprintf("%s/%s", namespace, repository)
		tags, err := client.ListTags(namespacedRepository)
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
	})

	// Root endpoint - provides basic information
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Get server information for API URL
		scheme, host := getServerInfo(r)
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
		if templates != nil {
			if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
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
	})

	// API root endpoint
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		scheme, host := getServerInfo(r)
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
	})

	// Index endpoint - shows available endpoints for a resource
	mux.HandleFunc("GET /api/v1/{namespace}/{repository}/{tag}", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		repository := r.PathValue("repository")
		tag := r.PathValue("tag")

		// Build base URL for this resource
		scheme, host := getServerInfo(r)
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
	GetFirstLayerReader(repository, tagName string) (*client.LayerInfo, error)
	ListTags(repository string) ([]string, error)
}
