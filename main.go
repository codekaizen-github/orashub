package main

import (
	"fmt"
	"os"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/client"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/internal/andrew"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server"
)

func main() {
	fmt.Println(andrew.Thing())
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		panic("REGISTRY environment variable is not set")
	}
	registry_username := os.Getenv("REGISTRY_USERNAME")
	if registry_username == "" {
		panic("REGISTRY_USERNAME environment variable is not set")
	}
	registry_password := os.Getenv("REGISTRY_PASSWORD")
	if registry_password == "" {
		panic("REGISTRY_PASSWORD environment variable is not set")
	}
	port := os.Getenv("PORT")
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
	// 		Username: os.Getenv("REGISTRY_USERNAME"),
	// 		Password: os.Getenv("REGISTRY_PASSWORD"),
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
