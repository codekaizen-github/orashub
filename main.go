package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/internal/andrew"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func main() {
	fmt.Println(andrew.Thing())

	registry := "ghcr.io"
	repository := "codekaizen-github/wp-github-gist-block"

	src, err := remote.NewRepository(fmt.Sprintf("%s/%s", registry, repository))
	if err != nil {
		panic(err)
	}
	// Note: The below code can be omitted if authentication is not required.
	src.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(registry, auth.Credential{
			Username: os.Getenv("REGISTRY_USERNAME"),
			Password: os.Getenv("REGISTRY_PASSWORD"),
		}),
	}

	dst := memory.New()
	ctx := context.Background()

	tagName := "package-latest"
	desc, err := oras.Copy(ctx, src, tagName, dst, tagName, oras.DefaultCopyOptions)
	if err != nil {
		panic(err) // Handle error
	}
	fmt.Println(desc.Digest)
}
