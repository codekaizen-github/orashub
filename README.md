# ORASHub

A tool for storing and retrieving files using OCI Registry As Storage (ORAS).

## Features

- Store files in any OCI-compatible registry (GitHub Container Registry, Docker Hub, etc.)
- Download files directly from the registry
- Stream content efficiently without needing temporary files
- Support for proper content types and download headers

## Installation

### Pre-built Binaries

You can download pre-built binaries for your platform from the [Releases](https://github.com/codekaizen-github/orashub/releases) page.

#### Linux (amd64)
```bash
curl -L https://github.com/codekaizen-github/orashub/releases/latest/download/orashub-linux-amd64 -o orashub
chmod +x orashub
```

#### macOS (amd64)
```bash
curl -L https://github.com/codekaizen-github/orashub/releases/latest/download/orashub-darwin-amd64 -o orashub
chmod +x orashub
```

#### Windows (amd64)
Download the `orashub-windows-amd64.exe` file from the releases page.

### Building from Source

If you have Go installed, you can build from source:

```bash
git clone https://github.com/codekaizen-github/orashub.git
cd orashub
go build -o orashub
```

## Usage

### Environment Variables

The application requires the following environment variables:

- `ORASHUB_REGISTRY`: The registry URL (e.g., `ghcr.io`)
- `ORASHUB_REGISTRY_USERNAME`: Username for registry authentication
- `ORASHUB_REGISTRY_PASSWORD`: Password for registry authentication
- `ORASHUB_PORT`: (Optional) Port to run the server on (default: 8080)

### Running the Server

```bash
export ORASHUB_REGISTRY=ghcr.io
export ORASHUB_REGISTRY_USERNAME=your-username
export ORASHUB_REGISTRY_PASSWORD=your-token
./orashub
```

The server will start on port 8080 (or the port specified in the `ORASHUB_PORT` environment variable).

### API Endpoints

#### Discovery Endpoints
- `GET /` - HTML welcome page with basic information
- `GET /api/v1` - API root showing available endpoint patterns
- `GET /api/v1/{namespace}/{repository}/{tag}` - Shows all endpoints for a specific plugin

#### Resource Endpoints
- `GET /api/v1/{namespace}/{repository}/{tag}/download` - Download the plugin
- `GET /api/v1/{namespace}/{repository}/{tag}/descriptor` - Get descriptor metadata
- `GET /api/v1/{namespace}/{repository}/{tag}/manifest` - Get manifest
- `GET /api/v1/{namespace}/{repository}/{tag}/annotations` - Get annotations

## License

[MIT License](LICENSE)
