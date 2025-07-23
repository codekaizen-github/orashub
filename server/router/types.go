package router

import "net/http"

type RouterInterface interface {
	SetupRoutes(mux *http.ServeMux)
}
