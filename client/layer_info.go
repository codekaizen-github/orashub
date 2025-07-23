package client

import "io"

// LayerInfo contains metadata about a layer
type LayerInfo struct {
	Reader    io.ReadCloser
	Filename  string
	MediaType string
	Size      int64
}

// Read implements io.Reader for the LayerInfo struct
func (l *LayerInfo) Read(p []byte) (n int, err error) {
	return l.Reader.Read(p)
}

// Close closes the underlying reader
func (l *LayerInfo) Close() error {
	return l.Reader.Close()
}

func (l *LayerInfo) GetFilename() string {
	return l.Filename
}

func (l *LayerInfo) GetMediaType() string {
	return l.MediaType
}

func (l *LayerInfo) GetSize() int64 {
	return l.Size
}
