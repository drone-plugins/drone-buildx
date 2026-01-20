# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **drone-buildx**, a Drone CI plugin that uses Docker-in-Docker to build and publish Docker images to container registries. It's a Go-based plugin that supports Docker Buildx with advanced features like multi-platform builds, cache management, and Buildx Bake mode.

## Development Commands

### Testing and Development
```bash
# Run tests with coverage
go test -cover ./...

# Run Go vet for static analysis
go vet ./...

# Build the main binary
go build -v -a -tags netgo -o release/linux/amd64/drone-docker ./cmd/drone-docker

# Cross-compile for different architectures
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -a -tags netgo -o release/drone-buildx-linux-amd64 ./cmd/drone-docker
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -a -tags netgo -o release/drone-buildx-darwin-arm64 ./cmd/drone-docker
```

### Build System
- Uses Go modules (`go.mod`)
- CI/CD runs on Drone CI with `.drone.yml` configuration
- Supports multiple architectures: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Uses buildkit for advanced Docker builds

## Code Architecture

### Core Components

1. **Main Entry Points** (`cmd/` directory):
   - `drone-docker`: Main Docker plugin executable
   - `drone-ecr`: Amazon ECR variant
   - `drone-gcr`: Google Container Registry variant  
   - `drone-acr`: Azure Container Registry variant
   - `drone-heroku`: Heroku variant

2. **Core Plugin Logic**:
   - `app.go`: CLI application setup and flag definitions
   - `docker.go`: Main plugin execution logic and Docker command orchestration
   - `buildx.go`: Docker Buildx builder management and setup
   - `plugin.yml`: Plugin configuration metadata

3. **Supporting Modules**:
   - `tags.go`: Docker tag generation and management
   - `daemon.go`: Docker daemon management
   - `cachemetrics.go`: Build cache metrics collection and parsing
   - `config/docker/`: Docker configuration file management

### Key Data Structures

- **Plugin**: Main plugin configuration containing all settings
- **Build**: Docker build parameters (tags, args, cache settings, etc.)
- **Builder**: Buildx builder configuration (driver, options, remote connections)
- **Daemon**: Docker daemon settings
- **Login**: Registry authentication parameters

### Important Features

1. **Buildx Bake Mode**: Supports Docker Buildx Bake for multi-target builds with HCL/JSON/Compose files
2. **Multi-Registry Support**: Can authenticate and push to multiple registries
3. **Cache Management**: Advanced caching with external cache backends (S3, registry, etc.)
4. **Push-Only Mode**: Can tag and push existing images without building
5. **Self-Hosted Buildkit**: Supports loading pre-built buildkit images for self-hosted runners

### Configuration Patterns

- Heavy use of environment variables with `PLUGIN_` prefix
- CLI flags map to environment variables via `EnvVar` field
- Support for both comma-separated and semicolon-separated options for complex values
- Fallback mechanisms for driver options and buildkit versions

## Testing

The codebase includes unit tests for:
- Tag generation (`tags_test.go`)
- Cache metrics parsing (`cachemetrics_test.go`)
- Docker configuration (`config/docker/config_test.go`)
- Output handling (`tee_test.go`)

Test files follow Go naming convention `*_test.go`.

## Dependencies

Key external dependencies:
- `github.com/drone-plugins/drone-plugin-lib`: Drone plugin utilities
- `github.com/urfave/cli`: Command-line interface framework
- `github.com/sirupsen/logrus`: Logging
- `github.com/aws/aws-sdk-go`: AWS integration for ECR/S3 cache

## Release Process

- Binary releases are created for multiple platforms and compressed with zstd
- Docker images are built for each registry variant (docker, ecr, gcr, acr, heroku)
- Release binaries are uploaded to GitHub releases and Harness storage