package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/internal/andrew"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server"
)

// Version information - will be set during build via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If version flag is set, print version info and exit
	if *versionFlag {
		fmt.Printf("WordPress Plugin Registry ORAS v%s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		return
	}

	fmt.Println(andrew.Thing())
	registry := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY")
	if registry == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY environment variable is not set")
	}
	registry_username := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME")
	if registry_username == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME environment variable is not set")
	}
	registry_password := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD")
	if registry_password == "" {
		panic("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD environment variable is not set")
	}
	port := os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}
	client := client.NewClient(
		registry,
		registry_username,
		registry_password,
	)
	router := server.InitializeRoutes(client)
	server.Serve(router, port) // Start the server with the initialized routes

	// registry := "ghcr.io"
	// repository := "codekaizen-github/wp-github-gist-block"

	// src, err := remote.NewRepository(fmt.Sprintf("%s/%s", registry, repository))
	// if err != nil {
	// 	panic(err)
	// }
	// // Note: The below code can be omitted if authentication is not required.
	// src.Client = &auth.Client{
	// 	Client: retry.DefaultClient,
	// 	Cache:  auth.NewCache(),
	// 	Credential: auth.StaticCredential(registry, auth.Credential{
	// 		Username: os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME"),
	// 		Password: os.Getenv("WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD"),
	// 	}),
	// }

	// dst := memory.New()
	// ctx := context.Background()

	// tagName := "package-latest"
	// desc, err := oras.Copy(ctx, src, tagName, dst, tagName, oras.DefaultCopyOptions)
	// if err != nil {
	// 	panic(err) // Handle error
	// }
	// fmt.Printf("Copied %s:%s to memory with descriptor: %+v\n", repository, tagName, desc)
}
