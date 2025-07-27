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

### Docker Image

A ready-to-use Docker image is available on GitHub Container Registry:

```bash
docker pull ghcr.io/codekaizen-github/orashub:latest
```

The image supports both amd64 and arm64 architectures. See the [Running with Docker](#2-using-docker) section for usage instructions.

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

- `ORASHUB_CONFIG_PATH`: Path to the configuration file (required)
- `ORASHUB_PORT`: (Optional) Port to run the server on (default: 8080)
- `ORASHUB_TEMPLATES_PATH`: (Optional) Path to HTML templates directory. If not set, a built-in fallback template will be used.

### Configuration File

The application uses a configuration file (`config.yaml`) to define registry connections and access policies. You must specify the path to this file using the `ORASHUB_CONFIG_PATH` environment variable.

#### Example Configuration

```yaml
# Registry credentials configuration
registries:
  - name: "${ORASHUB_REGISTRY}"  # Uses the value from ORASHUB_REGISTRY environment variable
    username: "${ORASHUB_REGISTRY_USERNAME}"
    password: "${ORASHUB_REGISTRY_PASSWORD}"

# Repository access policy
allowed_repositories:
  - "${ORASHUB_REGISTRY}/codekaizen-github/*"  # Wildcard pattern for repositories
blocked_repositories: []  # Empty list means no repositories are explicitly blocked
```

#### Configuration Sections

- **registries**: List of container registries and their credentials
  - **name**: Registry URL (e.g., `ghcr.io`)
  - **username**: Username for authentication (supports environment variable substitution)
  - **password**: Password for authentication (supports environment variable substitution)

- **allowed_repositories**: List of repository patterns that are allowed to be accessed
  - Supports wildcard patterns like `ghcr.io/username/*`
  - If empty, all repositories are allowed

- **blocked_repositories**: List of repository patterns that should be blocked
  - Takes precedence over allowed_repositories
  - If empty, no repositories are explicitly blocked

### Running ORASHub

There are three ways to run ORASHub:

#### 1. Using Pre-built Binaries

Download the appropriate binary for your platform from the releases page, then run:

```bash
# Set environment variables for registry access
export ORASHUB_REGISTRY=ghcr.io
export ORASHUB_REGISTRY_USERNAME=your-username
export ORASHUB_REGISTRY_PASSWORD=your-token

# Create a config file
cat > config.yaml << EOL
# Registry credentials configuration
registries:
  - name: "\${ORASHUB_REGISTRY}"
    username: "\${ORASHUB_REGISTRY_USERNAME}"
    password: "\${ORASHUB_REGISTRY_PASSWORD}"

# Repository access policy
allowed_repositories:
  - "\${ORASHUB_REGISTRY}/codekaizen-github/*"
blocked_repositories: []
EOL

# Optional: Create custom templates
# mkdir -p ./templates
# cp path/to/your/templates/*.html ./templates/

# Point to the config file and run
export ORASHUB_CONFIG_PATH=$(pwd)/config.yaml
export ORASHUB_PORT=8080
# Optional: specify custom templates path
# export ORASHUB_TEMPLATES_PATH=$(pwd)/templates
./orashub
```

#### 2. Using Docker

ORASHub is available as a Docker image on GitHub Container Registry. The image is multi-architecture and supports both amd64 and arm64 platforms.

```bash
# Create a configuration file on your host machine
mkdir -p ./config
cat > ./config/config.yaml << EOL
# Registry credentials configuration
registries:
  - name: "\${ORASHUB_REGISTRY}"
    username: "\${ORASHUB_REGISTRY_USERNAME}"
    password: "\${ORASHUB_REGISTRY_PASSWORD}"

# Repository access policy
allowed_repositories:
  - "\${ORASHUB_REGISTRY}/codekaizen-github/*"
blocked_repositories: []
EOL

# Optional: Create custom templates
# mkdir -p ./templates
# cp path/to/your/templates/*.html ./templates/

# Run the Docker container with environment variables and volume mount
docker run -d \
  -p 8080:8080 \
  -e ORASHUB_REGISTRY=ghcr.io \
  -e ORASHUB_REGISTRY_USERNAME=your-username \
  -e ORASHUB_REGISTRY_PASSWORD=your-token \
  -e ORASHUB_CONFIG_PATH=/app/config/config.yaml \
  # Optional: mount and specify custom templates
  # -e ORASHUB_TEMPLATES_PATH=/app/templates \
  # -v $(pwd)/templates:/app/templates \
  -v $(pwd)/config:/app/config \
  ghcr.io/codekaizen-github/orashub:latest
```

You can also use a specific version by replacing `latest` with a version tag like `v1.0.0`.

#### 3. Building from Source

If you have Go installed, you can build and run from source:

```bash
# Clone the repository
git clone https://github.com/codekaizen-github/orashub.git
cd orashub

# Set environment variables
export ORASHUB_REGISTRY=ghcr.io
export ORASHUB_REGISTRY_USERNAME=your-username
export ORASHUB_REGISTRY_PASSWORD=your-token

# Build the application
go build -o orashub

# Use the development config or create your own
export ORASHUB_CONFIG_PATH=$(pwd)/dev/config.yaml
export ORASHUB_PORT=8080
# Optional: specify custom templates path
# export ORASHUB_TEMPLATES_PATH=$(pwd)/dev/templates

# Run the application
./orashub
```

### API Endpoints

#### Discovery Endpoints
- `GET /` - HTML welcome page with basic information
- `GET /api/v1` - API root showing available endpoint patterns
- `GET /api/v1/{namespace}/{repository}/{tag}` - Shows all endpoints for a specific plugin

#### Resource Endpoints
- `GET /api/v1/{namespace}/{repository}/{tag}/download` - Download the plugin
- `GET /api/v1/{namespace}/{repository}/{tag}/descriptor` - Get descriptor metadata
- `GET /api/v1/{namespace}/{repository}/{tag}/manifest` - Get manifest

## License

[MIT License](LICENSE)
