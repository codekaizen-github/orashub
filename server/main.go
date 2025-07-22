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
