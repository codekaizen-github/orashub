package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
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

func NewClient(registry string, username string, password string) *Client {
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
	/*
		{
			"schemaVersion": 2,
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"artifactType": "application/vnd.unknown.artifact.v1",
			"config": {
				"mediaType": "application/vnd.oci.empty.v1+json",
				"digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
				"size": 2,
				"data": "e30="
			},
			"layers": [
				{
					"mediaType": "application/zip",
					"digest": "sha256:f6e90061e892e879fa597d7e73a0f6889754741c257eb4963fba45d350dbee87",
					"size": 125443236,
					"annotations": {
						"org.opencontainers.image.title": "plugin.zip"
					}
				}
			],
			"annotations": {
				"org.codekaizen-github.wordpress-plugin-registry-oras.plugin-metadata": "{\"contributors\":[\"The WordPress Contributors\"],\"donate\":\"\",\"tags\":[\"block\"],\"requires\":\"\",\"tested\":\"6.7\",\"stable\":\"0.1.0\",\"short_description\":\"Example block scaffolded with Create Block tool.\",\"sections\":{\"description\":\"\u003cp\u003eThis is the long description. No limit, and you can use Markdown (as well as in the following sections).\u003c\\/p\u003e\\n\u003cp\u003eFor backwards compatibility, if this section is missing, the full length of the short description will be used, and\\nMarkdown parsed.\u003c\\/p\u003e\",\"installation\":\"\u003cp\u003eThis section describes how to install the plugin and get it working.\u003c\\/p\u003e\\n\u003cp\u003ee.g.\u003c\\/p\u003e\\n\u003col\u003e\\n\u003cli\u003eUpload the plugin files to the \u003ccode\u003e\\/wp-content\\/plugins\\/wp-github-gist-block\u003c\\/code\u003e directory, or install the plugin through the WordPress plugins screen directly.\u003c\\/li\u003e\\n\u003cli\u003eActivate the plugin through the 'Plugins' screen in WordPress\u003c\\/li\u003e\\n\u003c\\/ol\u003e\",\"frequently asked questions\":\"\u003ch4\u003eA question that someone might have\u003c\\/h4\u003e\\n\u003cp\u003eAn answer to that question.\u003c\\/p\u003e\\n\u003ch4\u003eWhat about foo bar?\u003c\\/h4\u003e\\n\u003cp\u003eAnswer to foo bar dilemma.\u003c\\/p\u003e\",\"screenshots\":\"\u003col\u003e\\n\u003cli\u003eThis screen shot description corresponds to screenshot-1.(png|jpg|jpeg|gif). Note that the screenshot is taken from\\nthe \\/assets directory or the directory that contains the stable readme.txt (tags or trunk). Screenshots in the \\/assets\\ndirectory take precedence. For example, \u003ccode\u003e\\/assets\\/screenshot-1.png\u003c\\/code\u003e would win over \u003ccode\u003e\\/tags\\/4.3\\/screenshot-1.png\u003c\\/code\u003e\\n(or jpg, jpeg, gif).\u003c\\/li\u003e\\n\u003cli\u003eThis is the second screen shot\u003c\\/li\u003e\\n\u003c\\/ol\u003e\",\"changelog\":\"\u003ch4\u003e0.1.0\u003c\\/h4\u003e\\n\u003cul\u003e\\n\u003cli\u003eRelease\u003c\\/li\u003e\\n\u003c\\/ul\u003e\",\"arbitrary section\":\"\u003cp\u003eYou may provide arbitrary sections, in the same format as the ones above. This may be of use for extremely complicated\\nplugins where more information needs to be conveyed that doesn't fit into the categories of \u0026quot;description\u0026quot; or\\n\u0026quot;installation.\u0026quot; Arbitrary sections will be shown below the built-in sections outlined above.\u003c\\/p\u003e\"},\"readme\":true,\"name\":\"Wp Github Gist Block\",\"plugin_uri\":\"\",\"version\":\"0.1.0\",\"description\":\"Example block scaffolded with Create Block tool.\",\"author\":\"The WordPress Contributors\",\"author_profile\":\"\",\"text_domain\":\"wp-github-gist-block\",\"domain_path\":\"\",\"network\":false,\"plugin\":\".\\/wp-github-gist-block.php\",\"slug\":\"wp-github-gist-block\"}",
				"org.opencontainers.image.created": "2025-07-20T21:50:42Z"
			}
		}
	*/
}
func (c *Client) GetAnnotations(repository string, tagName string) (map[string]string, error) {
	desc, err := c.GetDescriptor(repository, tagName)
	if err != nil {
		return nil, err // Handle error
	}
	return desc.Annotations, nil
}

func (c *Client) GetFirstLayerReader(repository, tagName string) (io.ReadCloser, error) {
	manifestBytes, err := c.GetManifest(repository, tagName)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Layers []struct {
			Digest      string            `json:"digest"`
			Size        int64             `json:"size"`
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
	fmt.Printf("Fetching first layer with digest: %s\n", layerDigest)

	// Get the filename from the layer's annotations if available
	filename := "plugin.zip" // Default filename
	if manifest.Layers[0].Annotations != nil {
		if title, ok := manifest.Layers[0].Annotations["org.opencontainers.image.title"]; ok && title != "" {
			filename = title
		}
	}

	// Create a temporary file to store the layer
	tmpFile, err := os.CreateTemp("", "layer-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	// Create a file store pointing to the directory containing our temp file
	tmpDir := filepath.Dir(tmpFile.Name())
	tmpFileName := filepath.Base(tmpFile.Name())

	// Close and remove the temp file - we'll let the file store manage it
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	// Create a file store
	fs, err := file.New(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %v", err)
	}

	// Connect to the remote repository
	repo, err := c.GetRepository(repository)
	if err != nil {
		fs.Close()
		return nil, err
	}

	// Prepare the descriptor for the layer we want to fetch
	desc := v1.Descriptor{
		MediaType: "application/zip",
		Digest:    digest.Digest(layerDigest),
		Size:      manifest.Layers[0].Size,
		Annotations: map[string]string{
			"org.opencontainers.image.title": tmpFileName,
		},
	}

	// Now manually fetch and save the blob
	fmt.Println("Pulling blob to file:", tmpFile.Name())

	// Get the target artifact - we only want the specific layer
	blob, err := repo.Fetch(c.Context, desc)
	if err != nil {
		fs.Close()
		return nil, fmt.Errorf("failed to fetch blob: %v", err)
	}

	// Save the blob to our temp file
	err = fs.Push(c.Context, desc, blob)
	if err != nil {
		blob.Close()
		fs.Close()
		return nil, fmt.Errorf("failed to save blob: %v", err)
	}
	blob.Close()

	// Now open the saved file for reading
	localFile, err := os.Open(filepath.Join(tmpDir, tmpFileName))
	if err != nil {
		fs.Close()
		return nil, fmt.Errorf("failed to open saved file: %v", err)
	}

	// Return a ReadCloser that also closes the file store when done
	return &readCloserWithCleanup{
		ReadCloser: localFile,
		cleanup: func() {
			fs.Close()
			os.Remove(filepath.Join(tmpDir, tmpFileName))
		},
		filename: filename, // Store the original filename
	}, nil
}

// readCloserWithCleanup wraps an io.ReadCloser and performs additional cleanup when closed
type readCloserWithCleanup struct {
	io.ReadCloser
	cleanup  func()
	filename string
}

func (r *readCloserWithCleanup) Close() error {
	err := r.ReadCloser.Close()
	r.cleanup()
	return err
}

// GetFilename returns the original filename
func (r *readCloserWithCleanup) GetFilename() string {
	return r.filename
}
