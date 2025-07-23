package router

import (
	"fmt"
	"net/http"
	"os"
)

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
