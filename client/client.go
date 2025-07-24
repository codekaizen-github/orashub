// ListTags returns all tags for a given repository
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Client struct {
	AuthClient  *auth.Client
	Registry    string
	MemoryStore *memory.Store
	Context     context.Context
}

func NewClient(registry string, username string, password string) ClientInterface {
	dst := memory.New()
	ctx := context.Background()
	authClient := &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(registry, auth.Credential{
			Username: username,
			Password: password,
		}),
	}
	return &Client{
		AuthClient:  authClient,
		Registry:    registry,
		MemoryStore: dst,
		Context:     ctx,
	}
}

func (c *Client) GetRepository(repository string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", c.Registry, repository))
	if err != nil {
		return nil, err // Handle error
	}
	repo.Client = c.AuthClient
	return repo, nil
}

// GetRegistry returns the registry URL configured for this client
func (c *Client) GetRegistry() string {
	return c.Registry
}
func (c *Client) GetDescriptor(repository string, tagName string) (*v1.Descriptor, error) {
	src, err := c.GetRepository(repository)
	if err != nil {
		return nil, err // Handle error
	}

	desc, err := oras.Copy(c.Context, src, tagName, c.MemoryStore, tagName, oras.DefaultCopyOptions)
	if err != nil {
		return nil, err // Handle error
	}
	return &desc, nil
}
func (c *Client) GetManifest(repository string, tagName string) ([]byte, error) {
	desc, err := c.GetDescriptor(repository, tagName)
	if err != nil {
		return nil, err // Handle error
	}
	content, err := c.MemoryStore.Fetch(c.Context, *desc)
	if err != nil {
		return nil, err // Handle error
	}
	readContent, err := io.ReadAll(content)
	if err != nil {
		return nil, err // Handle error
	}
	return readContent, nil
}
func (c *Client) GetFirstLayerReader(repository, tagName string) (LayerInfoInterface, error) {
	manifestBytes, err := c.GetManifest(repository, tagName)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Layers []struct {
			Digest      string            `json:"digest"`
			Size        int64             `json:"size"`
			MediaType   string            `json:"mediaType"`
			Annotations map[string]string `json:"annotations"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, err
	}
	if len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("no layers found in manifest")
	}

	layerDigest := manifest.Layers[0].Digest

	// Get the filename from the layer's annotations if available
	filename := "plugin.zip" // Default filename
	if manifest.Layers[0].Annotations != nil {
		if title, ok := manifest.Layers[0].Annotations["org.opencontainers.image.title"]; ok && title != "" {
			filename = title
		}
	}

	// Connect to the remote repository
	repo, err := c.GetRepository(repository)
	if err != nil {
		return nil, err
	}

	// Prepare the descriptor for the layer we want to fetch.
	// Needed else you get mismatch Content-Length errors.
	desc := v1.Descriptor{
		MediaType: manifest.Layers[0].MediaType,
		Digest:    digest.Digest(layerDigest),
		Size:      manifest.Layers[0].Size,
	}

	// Fetch the blob directly - this returns an io.ReadCloser we can stream
	content, err := repo.Fetch(c.Context, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob: %v", err)
	}

	// Return our new LayerInfo with all metadata
	return &LayerInfo{
		Reader:    content,
		Filename:  filename,
		MediaType: manifest.Layers[0].MediaType,
		Size:      manifest.Layers[0].Size,
	}, nil
}

// ListTags returns all tags for a given repository
func (c *Client) ListTags(repository string) ([]string, error) {
	repo, err := c.GetRepository(repository)
	if err != nil {
		return nil, err
	}

	var tags []string
	err = repo.Tags(c.Context, "", func(receivedTags []string) error {
		tags = append(tags, receivedTags...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}
