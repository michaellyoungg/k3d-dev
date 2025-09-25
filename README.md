# DevEnv - Local Development Environment Tool

A unified command-line tool that orchestrates k3d, Helm, and Terraform to provide seamless local development environments for microservice applications.

## Overview

DevEnv abstracts away infrastructure complexity while maintaining flexibility and power, enabling developers to focus on building features rather than managing tooling.

### Key Features

- **Unified CLI**: Single interface for k3d and Helm
- **Environment Management**: Complete lifecycle management (init, up, stop, status)
- **Service Orchestration**: Dependency management and health checks
- **Flexible Sources**: Local development, container registries, and Helm charts
- **Cross-Platform**: Native binaries for macOS, Linux, and Windows
- **Developer Experience**: Rich CLI with autocompletion and helpful error messages

## Quick Start

### Prerequisites

DevEnv requires the following tools to be installed:

- [Docker](https://docs.docker.com/get-docker/) - Container runtime
- [k3d](https://k3d.io/stable/#installation) - Lightweight Kubernetes
- [Helm](https://helm.sh/docs/intro/install/) - Kubernetes package manager

### Installation

Currently in development. To install from source:

```bash
# Clone repository
git clone https://github.com/your-org/plat.git
cd plat

# Build and install
go build -o plat
sudo mv plat /usr/local/bin/
```

### Basic Usage

```bash
# Check system prerequisites
plat doctor

# Initialize new environment
plat init my-project

# Start environment
plat up

# Check status
plat status

# Stop environment
plat stop
```

## Development

### Development Workflow

```bash
# Clone repository
git clone https://github.com/your-org/plat.git
cd plat

# Build and test locally
go build -o plat
./plat doctor

# Run tests
go test ./...

# Format code
go fmt ./...
```

### Project Structure

```
.
├── .claude/                 # Documentation and decisions
├── cmd/                     # CLI command implementations
├── pkg/                     # Public packages
│   └── tools/              # External tool integrations
├── internal/               # Private packages
├── main.go                # Application entry point
└── go.mod                 # Go module definition
```

## Commands

### Environment Lifecycle

- `plat init` - Initialize new development environment
- `plat up` - Start environment and services
- `plat stop` - Stop environment
- `plat status` - Show environment status
- `plat doctor` - Check system prerequisites

### Service Management

- `plat deploy <service>` - Deploy/update specific service
- `plat scale <service>=<replicas>` - Scale service instances
- `plat logs [--follow] [--services <list>]` - View service logs
- `plat exec <service> <command>` - Execute command in service

### Configuration

- `plat config show` - Display current configuration
- `plat config edit` - Edit configuration interactively
- `plat config validate` - Validate configuration files

## Configuration

DevEnv uses YAML configuration files that support flexible service sources - local development, container registries, and Helm charts.

### Service Source Types

**Local Development** (build from source):
```yaml
services:
  - name: user-service
    source:
      type: local
      path: "../user-service"  # Relative or absolute path
      tag: "dev"               # Optional build tag
```

**Container Registry** (use published images):
```yaml  
services:
  - name: payment-service
    source:
      type: registry
      image: "ghcr.io/my-org/payment-service:v2.1.0"
```

**Helm Charts** (deploy existing charts):
```yaml
services:
  - name: postgres
    source:
      type: helm
    chart:
      repository: "https://charts.bitnami.com/bitnami"
      name: "postgresql"
      version: "12.12.10"
```

### Complete Example

```yaml
# .plat/config.yml
apiVersion: plat/v1
kind: Environment
metadata:
  name: e-commerce
spec:
  defaults:
    registry: "ghcr.io/my-org"
    helmRepository: "oci://ghcr.io/my-org/charts"
    chart: "microservice"

  cluster:
    name: ecommerce-local
    k3d:
      agents: 2
      ports:
        - "80:80@loadbalancer"

  services:
    # Local development
    - name: user-service
      source:
        type: local
        path: "../user-service"
      dependencies: ["postgres"]
      
    # Published service
    - name: payment-service
      source:
        type: registry
        image: "${defaults.registry}/payment-service:v2.1.0"
        
    # External database
    - name: postgres
      source:
        type: helm
      chart:
        repository: "https://charts.bitnami.com/bitnami"
        name: "postgresql"
```

### Multi-Repository Workflows

**Scenario 1**: All services from registry (new developer setup)
```yaml
services:
  - name: user-service
    source: {type: registry, image: "my-org/user-service:latest"}
  - name: payment-service  
    source: {type: registry, image: "my-org/payment-service:latest"}
```

**Scenario 2**: Local development of specific services
```yaml
services:
  - name: user-service
    source: {type: local, path: "../user-service"}      # Local development
  - name: payment-service
    source: {type: registry, image: "my-org/payment-service:v1.2.0"}  # Stable version
```

See `examples/config.yml` for a complete multi-service configuration.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and architecture documentation.

## License

MIT License - see [LICENSE](LICENSE) for details.