package client

import (
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type LayerInfoInterface interface {
	Close() error
	Read(p []byte) (n int, err error)
	GetFilename() string
	GetMediaType() string
	GetSize() int64
}

// ClientInterface defines the methods a client must implement
type ClientInterface interface {
	GetDescriptor(repository string, tagName string) (*v1.Descriptor, error)
	GetManifest(repository string, tagName string) ([]byte, error)
	GetFirstLayerReader(repository, tagName string) (LayerInfoInterface, error)
	ListTags(repository string) ([]string, error)
	GetRegistry() string
}
