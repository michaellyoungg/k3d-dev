# Decision Record: Service Configuration Design

**Date**: 2024-09-25
**Status**: Proposed
**Context**: Designing configuration system for multi-repo microservice development

## Problem Statement

Developers need a flexible configuration system that supports:
1. **Multi-repo patterns** - services across different repositories
2. **Mixed source types** - some services from registries, others from local development
3. **Default fallbacks** - sensible defaults when no custom config provided
4. **Local development override** - easy switching between registry and local sources

### Success Criteria
- [ ] Support monorepo and multi-repo development patterns (FR-022)
- [ ] Hierarchical configuration inheritance (FR-023) 
- [ ] Zero-configuration startup with sensible defaults
- [ ] Easy local development workflow switching
- [ ] Version control friendly configuration

### Constraints
- Must integrate with existing container and Helm registries
- Local paths should work across different development machines
- Configuration should be declarative and version-controllable

## Research Summary

### Existing Documentation Consulted
- [x] `02-requirements.md` - FR-022/023 multi-repo and hierarchical config requirements
- [x] `03-use-cases.md` - UC-001 zero-config setup, UC-002 daily development workflow
- [x] `04-ergonomics.md` - Configuration philosophy and layered approach
- [x] `05-architecture.md` - Configuration system architecture

### External Research
- **Industry Patterns**: Centralized Helm chart libraries with service-specific values
- **Repository Patterns**: Separate ops repository for shared configurations, CI files with services
- **Docker Compose Evolution**: Multi-service orchestration with override patterns
- **GitOps Practices**: Declarative configuration with environment-specific overlays

## Options Analysis

### Option 1: Single Configuration File with Service Sources
**Description**: YAML file defining all services with flexible source types

**Configuration Structure**:
```yaml
# .devenv/config.yml
apiVersion: devenv/v1
kind: Environment
metadata:
  name: my-project
spec:
  defaults:
    registry: "registry.local:5000"
    helmRepository: "oci://registry.local:5000/charts"
    
  services:
    - name: user-service
      source:
        type: local
        path: "../user-service"
      chart: "microservice"
      
    - name: payment-service  
      source:
        type: registry
        image: "my-org/payment-service:v1.2.3"
      chart:
        repository: "oci://registry.local:5000/charts" 
        name: "microservice"
        
    - name: postgres
      source:
        type: helm
      chart:
        repository: "https://charts.bitnami.com/bitnami"
        name: "postgresql"
```

**Pros**:
- Clear service-by-service control
- Supports all source types (local, registry, helm)
- Explicit configuration - no magic
- Easy to version control and share

**Cons**:
- Requires explicit configuration for each service
- Could become verbose for many services
- No automatic discovery of local repositories

**Effort**: Medium  
**Risk**: Low

### Option 2: Layered Configuration with Auto-Discovery
**Description**: Base configuration + local overrides with repository discovery

**Base Configuration** (`.devenv/defaults.yml`):
```yaml
services:
  - name: user-service
    image: "my-org/user-service:latest"
    chart: "microservice"
  - name: payment-service
    image: "my-org/payment-service:latest" 
    chart: "microservice"
```

**Local Override** (`.devenv/local.yml`):
```yaml
services:
  - name: user-service
    source:
      type: local
      path: "../user-service"
  - name: payment-service
    source:
      type: local
      path: "~/dev/payment-service"
```

**Pros**:
- Clean separation of defaults vs local overrides
- Local overrides not committed to repo
- Easy switching between modes
- Auto-discovery potential

**Cons**:
- More complex configuration merging
- Local overrides could get lost
- Need to maintain two config formats

**Effort**: High
**Risk**: Medium

### Option 3: Service Discovery with Convention over Configuration
**Description**: Auto-discover services based on directory structure and conventions

**Directory Structure**:
```
project/
├── .devenv/
│   └── config.yml          # Basic environment config
├── services/
│   ├── user-service/       # Auto-discovered
│   └── payment-service/    # Auto-discovered
└── devenv-overrides.yml    # Optional local overrides
```

**Pros**:
- Minimal configuration required
- Follows established conventions
- Easy onboarding for new developers
- Works well with monorepo patterns

**Cons**:
- Less flexible for complex scenarios
- Magic behavior could be confusing
- Doesn't handle multi-repo patterns well
- May not work for all team structures

**Effort**: Medium
**Risk**: High (assumptions about structure)

## Decision

**Chosen**: Option 1 - Single Configuration File with Service Sources

**Rationale**:

1. **Explicit over Implicit**: Aligns with our principle of clear, understandable configuration without magic behavior
2. **Multi-repo Support**: Directly addresses FR-022 by supporting services from different repositories via flexible paths
3. **Hierarchical Configuration**: Supports FR-023 through defaults section + service-specific overrides
4. **Established Pattern**: Follows industry practices seen in Docker Compose, Kubernetes, and Helm
5. **Version Control Friendly**: Single file can be committed, local paths can be relative

**Key Factors**:
- Explicit source types eliminate confusion about where services come from
- Defaults section provides sensible fallbacks while allowing granular control
- Relative paths work across development environments
- Configuration is self-documenting and discoverable

## Implementation Plan

### Phase 1: Core Configuration Structure
- [ ] Define YAML schema for service configuration
- [ ] Implement configuration loading and validation
- [ ] Support for local path and registry source types
- [ ] Basic defaults merging system
- **Validation**: Can load and parse service configurations
- **Timeline**: 2-3 days

### Phase 2: Source Type Implementations  
- [ ] Local directory source (build container from Dockerfile)
- [ ] Container registry source (pull existing images)
- [ ] Helm chart source (install charts directly)
- [ ] Image and chart caching strategies
- **Validation**: Can deploy services from all source types
- **Timeline**: 1 week

### Phase 3: Advanced Configuration Features
- [ ] Environment-specific overlays (.devenv/environments/dev.yml)
- [ ] Configuration templating and variable substitution
- [ ] Service dependency declarations
- [ ] Hot-reload when local source changes
- **Validation**: Complex multi-service scenarios work smoothly
- **Timeline**: 1 week

## Configuration Examples

### Basic Multi-Repo Setup
```yaml
# .devenv/config.yml
apiVersion: devenv/v1
kind: Environment
metadata:
  name: e-commerce
spec:
  defaults:
    registry: "ghcr.io/my-org"
    helmRepository: "oci://ghcr.io/my-org/charts"
    chart: "microservice"
    
  services:
    # Local development of user service
    - name: user-service
      source:
        type: local
        path: "../user-service"
      ports: [3000]
      
    # Use published payment service  
    - name: payment-service
      source:
        type: registry
        image: "${defaults.registry}/payment-service:v2.1.0"
      ports: [3001]
      
    # External database
    - name: postgres
      source:
        type: helm
      chart:
        repository: "https://charts.bitnami.com/bitnami"
        name: "postgresql"
      values:
        auth:
          database: "myapp"
```

### Development Workflow Examples

#### Scenario 1: New Developer Setup
```bash
# Clone main project repo
git clone https://github.com/my-org/e-commerce.git
cd e-commerce

# Initialize with default configuration
devenv init

# Start with all registry images
devenv up
```

#### Scenario 2: Local Feature Development  
```bash
# Clone service for local development
git clone https://github.com/my-org/user-service.git ../user-service

# Edit .devenv/config.yml to use local path for user-service
# (or use CLI helper)
devenv config set services.user-service.source.type=local
devenv config set services.user-service.source.path=../user-service

# Restart with local user service
devenv restart --services user-service
```

#### Scenario 3: Mixed Development
```yaml
# Some services local, others from registry
services:
  - name: user-service
    source: {type: local, path: "../user-service"}
  - name: payment-service  
    source: {type: local, path: "~/dev/payment-service"}
  - name: order-service
    source: {type: registry, image: "my-org/order-service:v1.5.0"}
  - name: notification-service
    source: {type: registry, image: "my-org/notification-service:latest"}
```

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Path resolution across platforms | Medium | Medium | Use Go's filepath package, relative paths from config location |
| Configuration becomes complex | Medium | High | Provide CLI helpers, validation, good examples |
| Local builds slow down startup | High | Medium | Implement build caching, parallel builds |
| Service interdependencies complex | High | High | Start simple, add dependency management incrementally |

## Future Considerations

- **Configuration Templates**: Jinja2-style templating for dynamic values
- **Environment Profiles**: Production-like configs for integration testing  
- **Service Mesh Integration**: Automatic service discovery and routing
- **CI/CD Integration**: Auto-update registry versions in config files
- **IDE Integration**: VS Code extensions for config editing and validation