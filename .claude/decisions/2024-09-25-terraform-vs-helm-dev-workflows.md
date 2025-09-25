# Decision Record: Terraform vs Helm for Development Workflows

**Date**: 2024-09-25
**Status**: Proposed
**Context**: Choosing tools for hot-reload rebuild/redeploy workflows in development

## Problem Statement

We need to design the development workflow for file changes that require image rebuilding:

1. **File Change** → Save in local development
2. **Build Process** → Rebuild Docker image 
3. **Registry Push** → Push to k3d local registry
4. **Deployment Update** → Restart Kubernetes deployment
5. **Validation** → Verify new version running

Should Terraform manage this development workflow, or should we use other tools?

### Success Criteria
- [ ] Sub-10 second rebuild-to-running cycle for code changes
- [ ] Reliable image push to k3d local registry  
- [ ] Seamless deployment restart without downtime
- [ ] Integration with file watching for automatic triggering
- [ ] Support for both compiled (Go) and interpreted (Node.js) languages

### Constraints
- Must work with k3d local development setup
- Should integrate with our service configuration system
- Local registry at localhost:5000 for k3d
- Development speed prioritized over production-ready processes

## Research Summary

### Existing Documentation Consulted
- [x] `06-technology-integration.md` - Current Terraform and Helm integration plans
- [x] `04-ergonomics.md` - Developer experience requirements for hot-reload
- [x] `02-requirements.md` - NFR-002: Service hot-reload time < 10 seconds

### External Research
- **Industry Consensus 2024**: Terraform for infrastructure, Helm for application deployment
- **Development Tools**: Skaffold, Tilt, DevSpace specifically built for k8s hot-reload
- **Terraform Limitations**: No native hot-reload, slow for frequent deployment updates
- **Helm Advantages**: Fast rollouts, rollbacks, templating for dynamic image tags

## Options Analysis

### Option 1: Terraform for Everything
**Description**: Use Terraform kubernetes provider for all workload management

```hcl
resource "kubernetes_deployment" "user_service" {
  metadata {
    name = "user-service"
  }
  spec {
    template {
      spec {
        container {
          image = var.user_service_image # Updated on each rebuild
        }
      }
    }
  }
}
```

**Pros**:
- Single tool for infrastructure and workloads
- Declarative state management
- Integration with existing Terraform modules

**Cons**:
- **Slow hot-reload**: Terraform plan/apply cycle too slow for development
- **No rollback support**: Can't easily revert broken deployments
- **Limited k8s features**: Terraform k8s provider has known limitations
- **Not designed for frequent updates**: Terraform optimized for infrequent infrastructure changes

**Effort**: High (fighting against tool design)
**Risk**: High (poor developer experience)

### Option 2: Helm for Development Workloads  
**Description**: Terraform for cluster setup, Helm for all application deployments

```yaml
# Helm chart values updated dynamically
image:
  repository: localhost:5000/user-service
  tag: dev-{{ .Values.buildHash }}
  pullPolicy: Always
```

```bash
# Hot-reload workflow
docker build -t localhost:5000/user-service:dev-abc123 .
docker push localhost:5000/user-service:dev-abc123
helm upgrade user-service ./chart --set image.tag=dev-abc123
```

**Pros**:
- **Fast deployment updates**: Helm optimized for k8s application lifecycle
- **Native rollbacks**: `helm rollback` for quick recovery
- **Template flexibility**: Dynamic image tags, environment-specific values
- **K8s native**: Full support for all Kubernetes resources

**Cons**:
- Two-tool workflow (Terraform + Helm)
- Need to manage Helm chart templates
- Coordination between tools required

**Effort**: Medium
**Risk**: Low (established pattern)

### Option 3: Integrated Development Tools (Skaffold/Tilt)
**Description**: Use specialized k8s development tools that handle entire pipeline

```yaml
# skaffold.yaml
apiVersion: skaffold/v4beta1
kind: Config
build:
  artifacts:
  - image: localhost:5000/user-service
    context: ../user-service
deploy:
  helm:
    releases:
    - name: user-service
      chartPath: ./charts/user-service
```

**Pros**:
- **Purpose-built for hot-reload**: Sub-second rebuild-to-running cycles
- **File watching**: Automatic rebuilds on changes
- **Multi-language support**: Optimized for Go, Node.js, Python, etc.
- **Local registry integration**: Built-in support for k3d/kind registries

**Cons**:
- Additional tool dependency
- Learning curve for team
- May conflict with our configuration system
- Less control over individual steps

**Effort**: Medium
**Risk**: Medium (tool integration complexity)

## Decision

**Chosen**: Option 2 - Helm for Development Workloads

**Rationale**:

1. **Tool Specialization**: Terraform excels at infrastructure provisioning, Helm excels at application deployment. Use each tool for its strengths.

2. **Development Speed**: Helm's `upgrade` command is designed for frequent application updates, while Terraform's plan/apply cycle is too slow for hot-reload workflows.

3. **Kubernetes Native**: Helm understands Kubernetes deployment patterns, rollouts, and rollbacks better than Terraform's k8s provider.

4. **Integration Possible**: We can still use Terraform to install Helm charts for infrastructure components, while using Helm directly for development workloads.

**Key Factors**:
- NFR-002 requires <10 second hot-reload - Terraform can't achieve this
- Industry best practice is infrastructure (Terraform) + applications (Helm)  
- Our service configuration can generate Helm values dynamically
- Helm rollback capabilities essential for development iteration

## Implementation Plan

### Phase 1: Helm Chart Templates
- [ ] Create base Helm chart template for microservice pattern
- [ ] Support dynamic image tags and local registry
- [ ] Integrate with our service configuration system
- [ ] Template generation from service configs
- **Validation**: Can deploy service via Helm with dynamic values
- **Timeline**: 2-3 days

### Phase 2: Hot-Reload Workflow
- [ ] Implement file watching for local source services
- [ ] Build pipeline: detect changes → build image → push to k3d registry
- [ ] Helm upgrade with new image tag
- [ ] Health check validation and rollback on failure
- **Validation**: <10 second file-change-to-running cycle
- **Timeline**: 1 week

### Phase 3: Terraform Integration
- [ ] Use Terraform for cluster infrastructure (ingress, monitoring)
- [ ] Use Terraform helm provider for infrastructure charts only
- [ ] Keep development workloads as direct Helm deployments
- [ ] Clear separation of concerns
- **Validation**: Infrastructure stable, applications hot-reloadable  
- **Timeline**: 3-4 days

## Workflow Design

### Development Hot-Reload Cycle

```bash
# Triggered by file change in ../user-service/
1. devenv watch detects change
2. docker build -t localhost:5000/user-service:dev-$(git-hash) ../user-service
3. docker push localhost:5000/user-service:dev-$(git-hash)
4. helm upgrade user-service ./charts/microservice \
   --set image.tag=dev-$(git-hash) \
   --set image.pullPolicy=Always
5. kubectl rollout status deployment/user-service
6. Health check validation
```

### Service Configuration Integration

```yaml
# .devenv/config.yml
services:
  - name: user-service
    source:
      type: local
      path: "../user-service"
    chart:
      template: "microservice"  # Uses base template
      values:
        service:
          port: 3000
        ingress:
          host: "api.localhost"
```

### Generated Helm Values

```yaml
# Generated by devenv from service config
image:
  repository: localhost:5000/user-service
  tag: dev-abc123
  pullPolicy: Always

service:
  port: 3000
  targetPort: 3000

ingress:
  enabled: true
  host: api.localhost
  paths: ["/users"]
```

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Helm chart complexity | Medium | Medium | Start with simple template, expand incrementally |
| Image build slowness | High | High | Implement multi-stage builds, build caching |
| Registry push failures | Medium | Medium | Retry logic, local registry health checks |
| Deployment rollback needs | High | Low | Helm native rollback, automated health checks |

## Future Considerations

- **Skaffold Integration**: Could layer Skaffold on top for even faster workflows
- **Build Caching**: Docker layer caching and multi-stage builds
- **Parallel Builds**: Build multiple changed services simultaneously  
- **IDE Integration**: VS Code extensions for one-click deploy
- **Production Parity**: Same Helm charts for dev and production with different values