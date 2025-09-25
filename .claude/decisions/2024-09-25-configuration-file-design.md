# Decision Record: Configuration File Design

**Date**: 2024-09-25
**Status**: Accepted
**Context**: Designing configuration system for MSC/Platform local development workflows

## Problem Statement

We need a configuration system that supports:
1. **Base config with team defaults** (committed to repo)
2. **Developer-specific local overrides** (gitignored capability declarations)
3. **CLI mode selection** (artifact vs local execution)
4. **Flexible repo structures** (different Dockerfile/chart locations)
5. **Low maintenance versioning** (avoid constant version updates)

### Success Criteria
- [ ] New developers can run `plat up` immediately (all artifacts)
- [ ] Developers can declare local sources without affecting team config
- [ ] CLI controls execution mode (artifact vs local)
- [ ] Support diverse repository structures with minimal configuration
- [ ] Base config requires minimal maintenance

## Configuration Architecture

### Layered Configuration Model

#### 1. Base Config (Team Shared)
```yaml
# .plat/config.yml (committed, team default)
name: platform-backend
services:
  - payment-api     # Defaults to :latest tag
  - user-api        # Defaults to :latest tag
  - order-api       # Defaults to :latest tag
  - postgres: helm  # Always Helm chart
```

**Purpose**: Define services and default artifact versions
**Location**: Committed to main repository
**Maintenance**: Minimal - uses `:latest` tags by default

#### 2. Local Source Declarations (Developer Specific)
```yaml
# .plat/local.yml (gitignored, declares local capabilities)
local_sources:
  payment-api: "../payment-service"           # Simple path
  user-api:                                   # Complex structure
    path: "~/dev/user-service-monorepo"
    dockerfile: "services/user/Dockerfile"
    context: "services/user"
    chart: "charts/user-api"
```

**Purpose**: Declare what services are available for local development
**Location**: Gitignored, developer-specific
**Content**: Path mappings and repo-specific overrides

#### 3. Runtime Mode (CLI Controlled)
```bash
# Execution mode controls which sources to use
plat config set mode local    # Default to local sources
plat config set mode artifact # Default to artifacts

# Per-command overrides
plat up --local               # Use local sources this time
plat up --local payment-api   # Only payment-api local
```

## Configuration Schema

### Base Configuration
```yaml
apiVersion: plat/v1
kind: Environment
metadata:
  name: string              # Environment name
  description: string       # Optional description

spec:
  defaults:
    registry: string        # MSC container registry
    domain: string          # Local development domain
    namespace: string       # Kubernetes namespace
    
  services:
    - name: string          # Service name (simple form)
    # OR
    - name: string          # Service name (complex form)
      version: string       # Explicit version override
      chart: string|object  # Helm chart specification
      ports: [int]          # Port mappings
      environment: {string: string}  # Environment variables
      dependencies: [string] # Service dependencies
```

### Local Source Configuration
```yaml
local_sources:
  service-name: path                    # Simple form
  # OR
  service-name:                         # Complex form
    path: string                        # Repository path
    dockerfile: string                  # Dockerfile path (optional)
    context: string                     # Build context (optional)
    chart: string                       # Chart path (optional)
```

## Union Type Implementation

### YAML Union Types for Flexibility
Services support both simple and complex forms:

```yaml
# Simple form (80% of use cases)
services:
  - payment-api
  - user-api

# Mixed form (flexibility when needed)
services:
  - payment-api                    # Simple
  - name: user-api                 # Complex
    version: "v1.6.0-beta"
    ports: [3000, 9229]
    environment:
      DEBUG: "true"
```

Local sources also support union types:

```yaml
local_sources:
  payment-api: "../payment-service"    # Simple path
  user-api:                           # Complex structure
    path: "~/dev/user-monorepo"
    dockerfile: "services/user/Dockerfile"
    context: "services/user"
```

## Convention-Based Defaults

### MSC Repository Conventions
When using simple path form, assume MSC standards:

```
service-repo/
├── Dockerfile              # Default dockerfile location
├── chart/                  # Default Helm chart location
│   ├── Chart.yaml
│   └── values.yaml
└── src/                    # Application source
```

### Override Patterns for Non-Standard Repos

#### Monorepo Structure
```yaml
local_sources:
  user-api:
    path: "../platform-monorepo"
    dockerfile: "services/user-api/Dockerfile"
    context: "services/user-api"
    chart: "charts/user-api"
```

#### Legacy Repository
```yaml
local_sources:
  legacy-service:
    path: "../legacy-payment"
    dockerfile: "build/Dockerfile.k8s"
    context: "app"
    chart: "deployment/helm-chart"
```

## Version Management Strategy

### Default to :latest Tags
- **Base config uses implicit `:latest`** to minimize maintenance
- **Explicit versions only when needed** for testing/debugging
- **CLI overrides for temporary version pinning**

```yaml
# Base config - low maintenance
services:
  - payment-api     # Implicitly :latest
  - user-api        # Implicitly :latest

# Runtime version override
$ plat up --version payment-api=v2.1.0 user-api=v1.6.0-rc
```

## CLI Integration

### Mode Management
```bash
# Set persistent mode
plat config set mode local
plat config set mode artifact
plat config show mode

# Temporary overrides
plat up --local                    # All available local sources
plat up --local payment-api        # Specific service local
plat up --version user-api=v1.6.0  # Specific version override
```

### Status and Validation
```bash
plat status
# Mode: local
# 📦 payment-api: local ✨ (../payment-service) [building...]
# 📦 user-api: artifact (latest) [running]
# 📦 postgres: helm (postgresql) [running]

plat doctor --local
# 📋 Local Source Validation
# ✅ payment-api: ../payment-service (standard structure)
# ⚠️  user-api: ~/dev/user-monorepo (custom dockerfile)
# ❌ frontend: ../frontend-app (chart not found)
```

### Auto-Discovery
```bash
plat init --scan-local
# 🔍 Scanning for local repositories...
# Found: ../payment-service (standard structure)
# Found: ../user-monorepo (detected: services/user/Dockerfile)
# Generated .plat/local.yml with detected sources
```

## Implementation Details

### Go Type Definitions
```go
type LocalSource struct {
    // Simple form: just a path string
    SimplePath string
    
    // Complex form: structured configuration
    Path       string `yaml:"path"`
    Dockerfile string `yaml:"dockerfile,omitempty"`
    Context    string `yaml:"context,omitempty"`
    Chart      string `yaml:"chart,omitempty"`
}

func (ls *LocalSource) UnmarshalYAML(node *yaml.Node) error {
    // Handle both string and object forms
    var simplePath string
    if err := node.Decode(&simplePath); err == nil {
        ls.SimplePath = simplePath
        return nil
    }
    
    type localSourceAlias LocalSource
    return node.Decode((*localSourceAlias)(ls))
}
```

### Configuration Loading Priority
1. Load base config (`.plat/config.yml`)
2. Load local sources (`.plat/local.yml`) if exists
3. Apply MSC defaults (registry, domain, etc.)
4. Resolve execution mode (persistent setting + CLI overrides)
5. Generate runtime configuration

## File Locations

```
project-repo/
├── .plat/
│   ├── config.yml          # Base config (committed)
│   ├── local.yml           # Local sources (gitignored)
│   └── .platconfig         # CLI settings (gitignored)
```

## Benefits Achieved

1. **Zero Configuration Start**: New developers run `plat up` immediately
2. **Flexible Local Development**: Declare local sources without team impact  
3. **Minimal Maintenance**: `:latest` tags reduce config churn
4. **Repository Flexibility**: Support diverse structures with escapes
5. **Clear Mode Separation**: Artifact vs local execution is explicit
6. **Progressive Complexity**: Simple cases stay simple, complex cases possible

## Migration Path

### Phase 1: Base Configuration
- Implement base config loading with service definitions
- Support `:latest` tag defaults
- CLI mode management

### Phase 2: Local Source Integration  
- Add local source declarations
- Implement union type parsing
- Path validation and auto-discovery

### Phase 3: Advanced Features
- Complex repository structure support
- Enhanced validation and error reporting
- Integration with hot-reload workflows

---

*This configuration design balances simplicity for common cases with flexibility for complex repository structures, while maintaining clear separation between team defaults and developer-specific overrides.*