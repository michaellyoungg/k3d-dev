# Tool Ergonomics and User Experience

## Command-Line Interface Design

### Core Design Principles

1. **Simplicity First**: Common workflows should be single commands
2. **Progressive Disclosure**: Advanced features discoverable but not overwhelming
3. **Consistency**: Predictable command patterns and flag usage
4. **Feedback**: Clear status updates and actionable error messages
5. **Composability**: Commands work well together in scripts and automation

### Primary Command Structure

```bash
devenv <command> [subcommand] [options] [targets]
```

#### Environment Lifecycle
```bash
# Initialize new environment
devenv init [--template <name>] [--force]

# Start environment (all services or specific ones)
devenv up [--services service1,service2] [--profile <name>]

# Stop environment 
devenv stop [--preserve-data] [--services service1,service2]

# Restart environment or specific services
devenv restart [--services service1,service2]

# Reset environment to clean state
devenv reset [--hard] [--preserve-volumes]

# Get environment status
devenv status [--detailed] [--services service1,service2]
```

#### Service Management
```bash
# Deploy/update specific services
devenv deploy <service> [--chart-version <version>] [--values <file>]

# Scale services
devenv scale <service>=<replicas> [service2=replicas]

# Port forward to services
devenv forward <service>:<local-port>:<service-port>

# Execute commands in service containers
devenv exec <service> [-- <command>]

# Access service shells
devenv shell <service>
```

#### Development Workflow
```bash
# Watch for code changes and hot reload
devenv watch [--services service1,service2]

# Attach debugger to running service
devenv debug <service> [--port <debug-port>]

# View aggregated logs
devenv logs [--follow] [--services service1,service2] [--since <time>]

# Run tests against environment
devenv test [--service <service>] [--suite <test-suite>]
```

#### Data Management
```bash
# Load test data
devenv data load [--dataset <name>] [--service <service>]

# Reset data to clean state
devenv data reset [--service <service>]

# Create data snapshot
devenv data snapshot <name>

# Restore from snapshot
devenv data restore <name>

# Import/export data
devenv data export [--format json|sql] [--output <file>]
devenv data import <file> [--service <service>]
```

#### Configuration Management
```bash
# View current configuration
devenv config show [--service <service>]

# Set configuration values
devenv config set <key>=<value>

# Edit configuration interactively
devenv config edit [--service <service>]

# Validate configuration
devenv config validate

# Share configuration
devenv config export [--output <file>]
devenv config import <file>
```

## User Experience Patterns

### First-Run Experience

1. **Prerequisites Check**: Verify k3d, helm, terraform, docker availability
2. **Interactive Setup**: Guide user through initial configuration choices
3. **Template Selection**: Offer common project templates (web app, API, microservices)
4. **Validation**: Test environment startup with minimal services

```bash
$ devenv init
✓ Checking prerequisites...
✓ Docker is running
✓ k3d v5.4.6 found
✓ helm v3.10.2 found  
✓ terraform v1.3.6 found

? Select a project template:
  > Microservices (API Gateway + 2-3 backend services)
    Monolithic Web App
    Data Pipeline
    Custom (empty configuration)

? Cluster name (default: devenv-local): 
? Enable monitoring stack (Prometheus/Grafana)? Yes

🚀 Creating your development environment...
✓ Cluster created
✓ Base services deployed
✓ Configuration saved to .devenv/config.yml

Run 'devenv up' to start your environment!
```

### Progress Feedback

Clear visual feedback for long-running operations:

```bash
$ devenv up
🏗️  Starting development environment...

✓ Creating k3d cluster 'devenv-local'
✓ Installing ingress controller
⚙️  Deploying services...
  ✓ database (postgres:13)
  ⏳ api-gateway (building image...)
  ⏳ user-service (pulling chart...)
  ⏳ order-service (waiting for dependencies...)

🌐 Environment ready! 
   Dashboard: http://localhost:8080
   API: http://api.localhost
   
📖 View logs: devenv logs --follow
🔧 Debug mode: devenv debug <service>
```

### Error Handling and Recovery

Actionable error messages with suggested fixes:

```bash
$ devenv up
❌ Failed to start environment

Error: Port 5432 already in use
┌─ Conflict detected
│  PostgreSQL is already running on port 5432
│
├─ Suggested fixes:
│  1. Stop existing PostgreSQL: brew services stop postgresql  
│  2. Use different port: devenv config set postgres.port=5433
│  3. Remove conflicting service: devenv config remove postgres
│
└─ Run 'devenv doctor' for more diagnostics

$ devenv doctor
🔍 Diagnosing environment health...

✓ Docker daemon running
✓ k3d cluster accessible  
❌ Port conflicts detected:
  - 5432 (postgres) used by brew service
  - 6379 (redis) used by local redis-server
⚠️  Disk space low (2GB available, recommend 5GB minimum)

💡 Run 'devenv doctor --fix' to resolve automatically
```

### Context-Aware Help

Dynamic help based on current environment state:

```bash
$ devenv help
# Context: No environment initialized
Available commands:
  init     Initialize new development environment
  help     Show this help message
  version  Show version information

Get started: devenv init

$ devenv help  # After init but not started
Available commands:
  up       Start development environment
  config   Manage environment configuration  
  help     Show this help message

Next step: devenv up

$ devenv help  # Environment running
Environment: devenv-local (3 services running)

Common commands:
  logs     View service logs
  status   Check service health
  restart  Restart services
  stop     Stop environment

Service commands:
  deploy   Deploy/update services
  exec     Execute commands in services
  debug    Attach debugger

Run 'devenv help <command>' for detailed usage
```

## Configuration Philosophy

### Convention Over Configuration

Default behaviors that work for 80% of use cases:

- Standard port assignments (3000, 3001, 3002...)
- Common service names (api, web, db, cache)
- Sensible resource limits
- Automatic service discovery
- Standard volume mounts for common frameworks

### Layered Configuration

Configuration inheritance from multiple sources:

1. **Tool defaults**: Sensible defaults for all options
2. **Global config**: User preferences (`~/.devenv/config.yml`)
3. **Project config**: Project-specific settings (`.devenv/config.yml`)
4. **Environment config**: Environment-specific overrides (`.devenv/environments/`)
5. **Runtime flags**: Command-line overrides

### Configuration Discoverability

```bash
# See all available configuration options
devenv config schema

# Find configuration for specific feature
devenv config search "port"
devenv config search "database"

# Show configuration source and precedence
devenv config explain postgres.port
# postgres.port = 5432
#   Source: .devenv/environments/dev.yml:15
#   Overrides: default (5432)
```

## Workflow Integration

### IDE Integration

Support for common development workflows:

- **VS Code**: Workspace configuration generation
- **IntelliJ**: Run configuration templates
- **Vim/Neovim**: Quickfix integration for logs/errors

### Git Integration

Environment configurations tracked alongside code:

```bash
# Environment changes detected
$ git status
modified: .devenv/config.yml
new file: .devenv/services/payment-service.yml

$ devenv config diff
# Shows configuration changes with clear impact explanation
```

### CI/CD Integration

Environment definitions usable in automated testing:

```bash
# Export environment for CI
devenv export --format docker-compose > docker-compose.test.yml

# Generate Kubernetes manifests
devenv export --format kubernetes --output ./k8s/
```