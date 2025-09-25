# System Architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    DevEnv CLI Tool                         │
├─────────────────────────────────────────────────────────────┤
│  Command Interface  │  Configuration  │  Plugin System     │
│  - CLI Parser       │  - YAML/JSON    │  - Service Types   │
│  - Command Router   │  - Validation   │  - Custom Actions  │
│  - Context Manager  │  - Templating   │  - Hooks           │
├─────────────────────────────────────────────────────────────┤
│                   Core Engine                              │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Environment     │ Service         │ Infrastructure  │    │
│  │ Manager         │ Orchestrator    │ Provider        │    │
│  │                 │                 │                 │    │
│  │ - Lifecycle     │ - Dependencies  │ - k3d Clusters  │    │
│  │ - State Mgmt    │ - Deployment    │ - Helm Charts   │    │
│  │ - Snapshots     │ - Health Checks │ - Terraform     │    │
│  └─────────────────┴─────────────────┴─────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                 Integration Layer                           │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │   k3d API   │  Helm CLI   │ Terraform   │   Docker    │  │
│  │             │             │     CLI     │     API     │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                Local Infrastructure                         │
│  ┌─────────────────────────────────────────────────────────┐│
│  │            k3d Kubernetes Cluster                      ││
│  │  ┌─────────────┬─────────────┬─────────────┐          ││
│  │  │  Services   │  Ingress    │  Storage    │          ││
│  │  │  - Apps     │  - Routes   │  - Volumes  │          ││
│  │  │  - DBs      │  - TLS      │  - PVCs     │          ││
│  │  │  - Queues   │  - LB       │  - Secrets  │          ││
│  │  └─────────────┴─────────────┴─────────────┘          ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### Command Interface Layer

#### CLI Parser
- **Responsibility**: Parse commands, flags, and arguments
- **Technology**: Go Cobra library for consistent CLI patterns
- **Features**:
  - Auto-completion support
  - Help generation
  - Flag validation and type conversion
  - Command aliasing

#### Command Router
- **Responsibility**: Route commands to appropriate handlers
- **Features**:
  - Middleware pipeline (auth, logging, validation)
  - Context propagation
  - Error handling and recovery
  - Plugin command registration

#### Context Manager
- **Responsibility**: Manage execution context and state
- **Features**:
  - Current environment detection
  - Configuration loading and merging
  - User session management
  - Debug/verbose mode control

### Configuration System

#### Configuration Schema
```yaml
# .devenv/config.yml
apiVersion: devenv/v1
kind: Environment
metadata:
  name: my-project
  version: "1.0"
spec:
  cluster:
    name: my-project-local
    k3d:
      agents: 2
      ports:
        - "80:80@loadbalancer"
        - "443:443@loadbalancer"
  
  services:
    - name: database
      type: postgresql
      chart: bitnami/postgresql
      values:
        auth:
          database: myapp
    
    - name: api
      type: application
      source: ./services/api
      chart: ./charts/api
      dependencies: [database]
      
  networking:
    ingress:
      class: traefik
      domain: localhost
    
  volumes:
    - name: db-data
      type: persistent
      size: 1Gi
```

#### Configuration Sources
1. **Built-in Defaults**: Sensible defaults for all options
2. **Global User Config**: `~/.devenv/config.yml`
3. **Project Config**: `.devenv/config.yml`
4. **Environment Configs**: `.devenv/environments/{name}.yml`
5. **Runtime Overrides**: Command-line flags and environment variables

### Core Engine

#### Environment Manager
```go
type EnvironmentManager interface {
    Create(config *EnvironmentConfig) error
    Start(name string, options StartOptions) error
    Stop(name string, options StopOptions) error
    Status(name string) (*EnvironmentStatus, error)
    Destroy(name string) error
    List() ([]*EnvironmentInfo, error)
}
```

**Responsibilities**:
- Environment lifecycle management
- State persistence and recovery
- Resource cleanup and garbage collection
- Environment isolation and naming

#### Service Orchestrator
```go
type ServiceOrchestrator interface {
    Deploy(service *Service, options DeployOptions) error
    Undeploy(service *Service) error
    Scale(service *Service, replicas int) error
    Restart(service *Service) error
    GetStatus(service *Service) (*ServiceStatus, error)
    GetLogs(service *Service, options LogOptions) (io.ReadCloser, error)
}
```

**Responsibilities**:
- Service deployment and updates
- Dependency resolution and ordering
- Health checking and recovery
- Log aggregation and streaming

#### Infrastructure Provider
```go
type InfrastructureProvider interface {
    CreateCluster(config *ClusterConfig) error
    DeleteCluster(name string) error
    ApplyManifests(cluster string, manifests []byte) error
    GetClusterStatus(name string) (*ClusterStatus, error)
}
```

**Responsibilities**:
- k3d cluster provisioning and management
- Terraform module execution
- Infrastructure state management
- Resource monitoring and limits

## Service Architecture

### Service Types

#### Application Services
- **Source**: Local source code or Git repository
- **Build**: Docker build or existing image
- **Deploy**: Helm chart with custom values
- **Features**: Hot reload, debugging, code injection

#### Database Services  
- **Source**: Helm charts (bitnami, etc.)
- **Deploy**: Standard configurations with persistence
- **Features**: Data seeding, backup/restore, version management

#### Infrastructure Services
- **Source**: Helm charts or Kubernetes manifests
- **Deploy**: System-level services (ingress, monitoring)
- **Features**: Automatic configuration, health monitoring

### Service Dependencies

```yaml
services:
  - name: database
    type: postgresql
    
  - name: cache
    type: redis
    
  - name: api
    type: application
    dependencies: [database, cache]
    readiness:
      path: /health
      initialDelaySeconds: 30
    
  - name: frontend
    type: application  
    dependencies: [api]
    ports:
      - name: http
        port: 3000
        targetPort: 3000
```

**Dependency Resolution**:
1. Topological sort of service dependency graph
2. Parallel deployment of independent services
3. Health check validation before dependent service startup
4. Graceful rollback on dependency failures

## Data Architecture

### State Management

#### Environment State
```go
type EnvironmentState struct {
    Name        string                 `json:"name"`
    Status      EnvironmentStatus      `json:"status"`
    Services    map[string]*ServiceState `json:"services"`
    Cluster     *ClusterState          `json:"cluster"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}
```

**Storage**: Local SQLite database (`~/.devenv/state.db`)
**Features**:
- Atomic state updates
- State history and rollback
- Cross-environment isolation
- Background state synchronization

#### Configuration Storage
- **Project Config**: Versioned in project repository
- **User Config**: Local user directory
- **Generated Config**: Temporary/cache directory
- **Secrets**: OS keychain integration

### Data Persistence

#### Volume Management
```yaml
volumes:
  - name: postgres-data
    type: persistent
    size: 1Gi
    storageClass: local-path
    
  - name: app-cache
    type: tmpfs
    size: 512Mi
    
  - name: shared-config
    type: configmap
    source: ./config/
```

**Volume Types**:
- **Persistent**: Survives environment restarts
- **Ephemeral**: Cleanup on environment stop  
- **Host Mount**: Direct host directory access
- **ConfigMap/Secret**: Configuration injection

## Integration Architecture

### Tool Integration

#### k3d Integration
```go
type K3dProvider struct {
    client k3d.Client
    config *K3dConfig
}

func (p *K3dProvider) CreateCluster(config *ClusterConfig) error {
    cluster := &k3d.Cluster{
        Name: config.Name,
        Agents: config.Agents,
        Servers: 1,
        Image: config.Image,
        Network: config.Network,
        Ports: config.Ports,
    }
    return p.client.CreateCluster(cluster)
}
```

#### Helm Integration
```go
type HelmProvider struct {
    client helm.Client
    config *HelmConfig
}

func (p *HelmProvider) InstallChart(release *Release) error {
    chart := &helm.Chart{
        Name: release.Chart,
        Version: release.Version,
        Repository: release.Repository,
    }
    
    return p.client.Install(release.Name, chart, release.Values)
}
```

#### Terraform Integration
```go
type TerraformProvider struct {
    workspace string
    config *TerraformConfig
}

func (p *TerraformProvider) Apply(module *Module) error {
    cmd := exec.Command("terraform", "apply", "-auto-approve")
    cmd.Dir = module.Path
    cmd.Env = append(os.Environ(), module.Variables...)
    
    return cmd.Run()
}
```

### Plugin Architecture

#### Plugin Interface
```go
type Plugin interface {
    Name() string
    Version() string
    Init(config PluginConfig) error
    
    // Optional interfaces
    ServiceProvider   // Provides custom service types
    CommandProvider   // Provides additional CLI commands
    HookProvider      // Provides lifecycle hooks
}
```

#### Plugin Discovery
- **Built-in Plugins**: Compiled into main binary
- **Local Plugins**: `~/.devenv/plugins/` directory
- **Project Plugins**: `.devenv/plugins/` directory
- **Remote Plugins**: Downloaded from registries

## Security Architecture

### Isolation
- **Network**: Kubernetes namespaces and network policies
- **Filesystem**: Container filesystem isolation
- **Process**: Container security contexts
- **Resource**: CPU/memory limits and quotas

### Secrets Management
```yaml
secrets:
  - name: database-credentials
    type: kubernetes-secret
    data:
      username: postgres
      password: ${POSTGRES_PASSWORD}
      
  - name: api-keys
    type: external-secret
    provider: vault
    path: secret/api-keys
```

**Features**:
- OS keychain integration for user secrets
- Environment variable injection
- External secret provider integration
- Automatic secret rotation support