package router

import "net/http"

// RouterInterface defines the methods a router must implement
type RouterInterface interface {
	SetupRoutes(mux *http.ServeMux)
}

// RegistryRouter is a router that handles multiple registry clients
type RegistryRouter struct {
	Routers map[string]RouterInterface // Map of registry name to router
}
