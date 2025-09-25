# Technology Integration Guide

## k3d Integration

### Cluster Management

#### Cluster Configuration
```yaml
# .devenv/config.yml
cluster:
  name: myapp-local
  k3d:
    image: rancher/k3s:v1.25.4-k3s1
    servers: 1
    agents: 2
    ports:
      - "80:80@loadbalancer"
      - "443:443@loadbalancer" 
      - "5432:5432@agent:0"  # Direct database access
    volumes:
      - "/tmp/devenv-storage:/var/lib/rancher/k3s/storage@all"
    registries:
      create: true
      config: |
        mirrors:
          "localhost:5000":
            endpoint:
              - "http://localhost:5000"
    options:
      - --disable=traefik  # Use custom ingress
      - --disable=metrics-server
```

#### Advanced k3d Features
- **Multi-server clusters**: HA control plane for testing
- **Custom CNI**: Calico, Flannel configuration
- **Registry integration**: Local registry for image caching
- **Load balancer**: Direct port access for debugging
- **Host volume mounts**: Persistent storage across recreations

### Network Configuration

#### Service Exposure Patterns
```yaml
services:
  - name: api
    networking:
      ports:
        - name: http
          port: 3000
          nodePort: 30300  # Direct k3d access
        - name: debug
          port: 9229
          hostPort: 9229   # Host debugging
          
  - name: database
    networking:
      type: ClusterIP  # Internal only
      ports:
        - port: 5432
      
  - name: frontend
    networking:
      ingress:
        enabled: true
        host: app.localhost
        paths:
          - path: /
            pathType: Prefix
```

## Helm Integration

### Chart Management

#### Built-in Chart Repository
```yaml
# Default chart sources
repositories:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami
    
  - name: prometheus-community
    url: https://prometheus-community.github.io/helm-charts
    
  - name: local
    url: file://.devenv/charts  # Local chart development
```

#### Service Chart Templates
```yaml
# .devenv/services/api/helm-values.yml
apiVersion: devenv/v1
kind: ServiceChart
metadata:
  name: api-service
spec:
  chart: 
    name: local/microservice  # Custom chart template
    version: "1.0.0"
  
  values:
    image:
      repository: "{{.Service.Image}}"
      tag: "{{.Service.Tag}}"
      pullPolicy: IfNotPresent
    
    service:
      port: "{{.Service.Port}}"
      
    ingress:
      enabled: true
      host: "{{.Service.Name}}.{{.Environment.Domain}}"
    
    autoscaling:
      enabled: false
      minReplicas: 1
      maxReplicas: 3
    
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "512Mi" 
        cpu: "500m"
```

### Chart Development Workflow

#### Local Chart Development
```bash
# Create new chart template
devenv chart create microservice --type application

# Validate chart locally
devenv chart lint ./charts/microservice

# Test chart with different values
devenv chart test microservice --values test-values.yml

# Package and install to local repo
devenv chart package ./charts/microservice --destination .devenv/charts/
```

#### Chart Versioning Strategy
- **Development**: Use `latest` or `dev` tags during active development
- **Testing**: Pin to specific versions for integration tests  
- **Production**: Always use immutable, semantic versions

### Values Management

#### Hierarchical Values
```yaml
# Global values (.devenv/values.yml)
global:
  domain: localhost
  registry: localhost:5000
  monitoring: true
  
# Environment values (.devenv/environments/dev.yml)  
services:
  api:
    replicas: 1
    debug: true
    
# Service values (.devenv/services/api/values.yml)
image:
  tag: latest
resources:
  limits:
    memory: 1Gi
```

## Terraform Integration

### Infrastructure as Code

#### Module Structure
```
.devenv/terraform/
├── main.tf                 # Root module
├── variables.tf           # Input variables  
├── outputs.tf             # Output values
├── modules/
│   ├── monitoring/        # Prometheus/Grafana setup
│   ├── logging/           # ELK/Loki setup
│   ├── database/          # Database cluster
│   └── networking/        # Ingress/DNS setup
└── environments/
    ├── dev.tfvars
    ├── staging.tfvars
    └── prod.tfvars
```

#### Service Infrastructure
```hcl
# .devenv/terraform/modules/service/main.tf
resource "kubernetes_namespace" "service" {
  metadata {
    name = var.service_name
    labels = {
      "app.kubernetes.io/name" = var.service_name
      "devenv.local/managed-by" = "devenv"
    }
  }
}

resource "kubernetes_secret" "service_config" {
  metadata {
    name = "${var.service_name}-config"
    namespace = kubernetes_namespace.service.metadata[0].name
  }
  
  data = var.service_secrets
}

resource "kubernetes_config_map" "service_config" {
  metadata {
    name = "${var.service_name}-config" 
    namespace = kubernetes_namespace.service.metadata[0].name
  }
  
  data = var.service_config
}
```

#### External Resource Integration
```hcl
# External database for shared services
resource "docker_container" "postgres" {
  count = var.external_database ? 1 : 0
  
  image = "postgres:13"
  name  = "${var.environment_name}-postgres"
  
  ports {
    internal = 5432
    external = var.postgres_port
  }
  
  env = [
    "POSTGRES_DB=${var.postgres_db}",
    "POSTGRES_USER=${var.postgres_user}",
    "POSTGRES_PASSWORD=${var.postgres_password}"
  ]
  
  volumes {
    host_path      = "${var.data_dir}/postgres"
    container_path = "/var/lib/postgresql/data"
  }
}
```

### State Management

#### Local State Backend
```hcl
terraform {
  backend "local" {
    path = ".devenv/terraform/terraform.tfstate"
  }
  
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.16"
    }
    helm = {
      source  = "hashicorp/helm" 
      version = "~> 2.8"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.23"
    }
  }
}
```

## Docker Integration

### Image Management

#### Development Images
```yaml
services:
  - name: api
    build:
      context: ./services/api
      dockerfile: Dockerfile.dev  # Development-optimized
      target: development
      args:
        NODE_ENV: development
    volumes:
      - ./services/api:/app  # Source code mounting
      - /app/node_modules    # Preserve node_modules
```

#### Multi-stage Builds
```dockerfile
# Dockerfile.dev
FROM node:18-alpine AS base
WORKDIR /app
COPY package*.json ./
RUN npm ci

FROM base AS development
ENV NODE_ENV=development
RUN npm install --only=development
COPY . .
EXPOSE 3000 9229
CMD ["npm", "run", "dev"]

FROM base AS production  
ENV NODE_ENV=production
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

### Registry Integration

#### Local Registry
```bash
# Automatically managed by devenv
devenv registry start  # Starts localhost:5000 registry

# Automatic image pushing
devenv build api --push  # Builds and pushes to local registry

# Image caching for faster rebuilds
devenv cache pull --services api,frontend
```

## Monitoring Integration

### Observability Stack

#### Prometheus + Grafana
```yaml
# .devenv/monitoring.yml
monitoring:
  prometheus:
    enabled: true
    chart: prometheus-community/kube-prometheus-stack
    values:
      grafana:
        adminPassword: admin
        ingress:
          enabled: true
          hosts: [grafana.localhost]
      
      prometheus:
        ingress:
          enabled: true
          hosts: [prometheus.localhost]
        
        additionalScrapeConfigs:
          - job_name: 'devenv-services'
            kubernetes_sd_configs:
              - role: pod
                namespaces:
                  names: [default]
```

#### Application Metrics
```yaml
services:
  - name: api
    monitoring:
      metrics:
        enabled: true
        path: /metrics
        port: 3001
      
      alerts:
        - name: HighErrorRate
          expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        
        - name: HighLatency  
          expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
```

### Logging Integration

#### Centralized Logging
```yaml
logging:
  driver: loki
  config:
    endpoint: http://loki.localhost
    
  retention: 7d
  
  services:
    - name: api
      level: debug
      format: json
      
    - name: database
      level: warn
      slowQueries: true
```

## Development Workflow Integration

### Hot Reload Implementation

#### File Watching
```yaml
services:
  - name: frontend
    development:
      watch:
        enabled: true
        paths: [src/, public/]
        ignore: [node_modules/, .git/]
        
      reload:
        command: npm run dev
        signal: SIGHUP
        
      sync:
        - source: ./src
          target: /app/src
        - source: ./public  
          target: /app/public
```

#### Language-Specific Integration
```yaml
# Node.js with nodemon
services:
  - name: api
    development:
      runtime: nodejs
      watch:
        tool: nodemon
        config: nodemon.json
        
# Python with watchdog        
  - name: ml-service
    development:
      runtime: python
      watch:
        tool: watchdog
        command: python app.py
        
# Go with air
  - name: gateway
    development:
      runtime: golang
      watch:
        tool: air
        config: .air.toml
```

### Debugging Integration

#### Remote Debugging
```yaml
services:
  - name: api
    development:
      debug:
        enabled: true
        port: 9229  # Node.js debug port
        
      ports:
        - name: debug
          port: 9229
          hostPort: 9229  # Direct access from IDE
```

#### IDE Configuration Generation
```bash
# Generate VS Code launch configurations
devenv ide vscode --output .vscode/

# Generate IntelliJ run configurations  
devenv ide intellij --output .idea/runConfigurations/
```