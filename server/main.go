package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codekaizen-github/orashub/server/logger"
	"github.com/codekaizen-github/orashub/server/policy"
	"github.com/codekaizen-github/orashub/server/router"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// LogLevelError only logs errors
	LogLevelError LogLevel = iota
	// LogLevelWarn logs warnings and errors
	LogLevelWarn
	// LogLevelInfo logs info, warnings and errors
	LogLevelInfo
	// LogLevelDebug logs debug, info, warnings and errors
	LogLevelDebug
	// LogLevelTrace logs everything including detailed trace information
	LogLevelTrace
)

// Version information - will be set during build via ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// LoadTemplates loads all templates from the templates directory
func LoadTemplates(templatesPath string) (*template.Template, error) {
	templates, err := template.ParseGlob(filepath.Join(templatesPath, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("error loading templates: %v", err)
	}
	return templates, nil
}

// CreateFallbackTemplate creates an in-memory template with the default HTML content
func CreateFallbackTemplate() *template.Template {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>ORASHub</title>
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
    <h1>ORASHub</h1>
    <p>A service for storing and retrieving files using OCI Registry As Storage (ORAS).</p>

    <h2>API Access</h2>
    <p>The API is available at: <code>{{.ApiURL}}</code></p>
    <a href="{{.ApiURL}}" class="api-link">Explore the API</a>

    <h2>Documentation</h2>
    <p>For more information, please refer to the <a href="https://github.com/codekaizen-github/orashub">GitHub repository</a>.</p>
</body>
</html>`

	// Parse the template
	t, err := template.New("index.html").Parse(tmpl)
	if err != nil {
		// Using standard log here is fine since this is initialization code
		// and the logger might not be fully set up yet
		log.Printf("Error parsing fallback template: %v", err)
		return nil
	}
	return t
}

// Start initializes and starts the server, handling version flags
func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	logLevelFlag := flag.String("log-level", "", "Set log level (error, warn, info, debug, trace)")
	flag.Parse()

	// If version flag is set, print version info and exit
	if *versionFlag {
		fmt.Printf("ORASHub v%s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", Date)
		return
	}

	// Create default logger with INFO level
	appLogger := logger.NewDefaultLogger(logger.LogLevelInfo)

	// Set log level from flag or environment variable
	if *logLevelFlag != "" {
		if err := appLogger.SetLevelFromString(*logLevelFlag); err != nil {
			log.Printf("Warning: %v", err)
		}
	} else {
		// Check for environment variable
		envLogLevel := os.Getenv("ORASHUB_LOG_LEVEL")
		if envLogLevel != "" {
			if err := appLogger.SetLevelFromString(envLogLevel); err != nil {
				log.Printf("Warning: %v", err)
			}
		}
	}

	// Initialize and start the server
	Initialize(appLogger)
}

// Initialize creates a new client and server based on environment variables
func Initialize(appLogger logger.Logger) {
	// Get port with default fallback
	port := os.Getenv("ORASHUB_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}

	// Get config file path with default fallback
	configPath := os.Getenv("ORASHUB_CONFIG_PATH")
	if configPath == "" {
		appLogger.Error("ORASHUB_CONFIG_PATH environment variable is not set")
		log.Fatalf("ORASHUB_CONFIG_PATH environment variable is not set")
	}

	// Load configuration file
	config, err := policy.LoadConfig(configPath)
	if err != nil {
		appLogger.Error("Error loading configuration: %v", err)
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Get image policy from the configuration
	imagePolicy := config.GetImagePolicy()

	// Load templates if path is provided
	templatesPath := os.Getenv("ORASHUB_TEMPLATES_PATH")
	var templates *template.Template

	if templatesPath != "" {
		var err error
		templates, err = LoadTemplates(templatesPath)
		if err != nil {
			appLogger.Warn("Error loading templates from %s: %v", templatesPath, err)
			appLogger.Warn("Using fallback template instead")
			templates = CreateFallbackTemplate()
		} else {
			appLogger.Info("Loaded templates from: %s", templatesPath)
		}
	} else {
		appLogger.Info("ORASHUB_TEMPLATES_PATH not set, using fallback template")
		templates = CreateFallbackTemplate()
	}

	// Create API manager
	manager := router.NewApiManager(config, imagePolicy, templates, appLogger)

	// Create mux and set up routes using the manager
	mux := http.NewServeMux()
	manager.SetupRoutes(mux)

	// Wrap mux with logging middleware
	loggedMux := logger.LoggingMiddleware(appLogger, mux)

	// Start the server with the configured mux
	Serve(loggedMux, port, appLogger)
}

// Entry point of the program
func Serve(handler http.Handler, port string, appLogger logger.Logger) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}
	appLogger.Info("Server listening on port %s", port)
	appLogger.Error("Server stopped: %v", server.ListenAndServe()) // Run the http server
}
