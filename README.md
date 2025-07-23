# WordPress Plugin Registry ORAS

A tool for storing and retrieving WordPress plugins using OCI Registry As Storage (ORAS).

## Features

- Store WordPress plugins in any OCI-compatible registry (GitHub Container Registry, Docker Hub, etc.)
- Download plugins directly from the registry
- Stream content efficiently without needing temporary files
- Support for proper content types and download headers

## Installation

### Pre-built Binaries

You can download pre-built binaries for your platform from the [Releases](https://github.com/codekaizen-github/wordpress-plugin-registry-oras/releases) page.

#### Linux (amd64)
```bash
curl -L https://github.com/codekaizen-github/wordpress-plugin-registry-oras/releases/latest/download/wordpress-plugin-registry-oras-linux-amd64 -o wordpress-plugin-registry-oras
chmod +x wordpress-plugin-registry-oras
```

#### macOS (amd64)
```bash
curl -L https://github.com/codekaizen-github/wordpress-plugin-registry-oras/releases/latest/download/wordpress-plugin-registry-oras-darwin-amd64 -o wordpress-plugin-registry-oras
chmod +x wordpress-plugin-registry-oras
```

#### Windows (amd64)
Download the `wordpress-plugin-registry-oras-windows-amd64.exe` file from the releases page.

### Building from Source

If you have Go installed, you can build from source:

```bash
git clone https://github.com/codekaizen-github/wordpress-plugin-registry-oras.git
cd wordpress-plugin-registry-oras
go build -o wordpress-plugin-registry-oras
```

## Usage

### Environment Variables

The application requires the following environment variables:

- `WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY`: The registry URL (e.g., `ghcr.io`)
- `WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME`: Username for registry authentication
- `WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD`: Password for registry authentication
- `WORDPRESS_PLUGIN_REGISTRY_ORAS_PORT`: (Optional) Port to run the server on (default: 8080)

### Running the Server

```bash
export WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY=ghcr.io
export WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_USERNAME=your-username
export WORDPRESS_PLUGIN_REGISTRY_ORAS_REGISTRY_PASSWORD=your-token
./wordpress-plugin-registry-oras
```

The server will start on port 8080 (or the port specified in the `WORDPRESS_PLUGIN_REGISTRY_ORAS_PORT` environment variable).

### API Endpoints

- `GET /api/v1/{namespace}/{repository}/{tag}/download` - Download the plugin
- `GET /api/v1/{namespace}/{repository}/{tag}/descriptor` - Get descriptor metadata
- `GET /api/v1/{namespace}/{repository}/{tag}/manifest` - Get manifest
- `GET /api/v1/{namespace}/{repository}/{tag}/annotations` - Get annotations

## License

[MIT License](LICENSE)
