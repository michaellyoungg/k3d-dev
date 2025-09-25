# Decision Record: Remove Terraform from Toolchain

**Date**: 2024-09-25
**Status**: Accepted
**Context**: Simplifying toolchain by eliminating unnecessary Terraform layer

## Problem Statement

Our planned toolchain included k3d + Terraform + Helm, but analysis reveals Terraform adds no value in local development context where k3d provides the cluster and all subsequent resources are Kubernetes-native.

### Success Criteria
- [ ] Simpler toolchain with fewer moving parts
- [ ] Faster development workflows
- [ ] Reduced learning curve for developers
- [ ] Maintain all required functionality

## Analysis

### What Terraform Was Planned For
- Infrastructure provisioning → **k3d already handles this**
- Supporting infrastructure → **Just Kubernetes resources, Helm handles better**
- Storage volumes → **Kubernetes PVCs, not cloud storage**
- Networking → **Kubernetes Services/Ingress, not cloud networking**

### What k3d + Helm Provides
- ✅ Cluster creation and management
- ✅ Local container registry
- ✅ All Kubernetes resources (deployments, services, ingress)
- ✅ Application lifecycle management
- ✅ Infrastructure components via Helm charts
- ✅ Fast hot-reload workflows

## Decision

**Remove Terraform entirely** from the devenv toolchain.

**New Simplified Toolchain**: k3d → Helm → Services

**Rationale**:
1. **No Added Value**: Everything Terraform would manage is better handled by Helm
2. **Complexity Reduction**: One fewer tool to learn, configure, and maintain
3. **Speed**: Direct Helm deployments faster than Terraform → Helm chain
4. **Native Kubernetes**: Helm understands Kubernetes patterns better than Terraform k8s provider

## Implementation Changes

### Remove Terraform Components
- [ ] Remove Terraform interfaces from `pkg/tools/`
- [ ] Remove Terraform configuration from service config types
- [ ] Remove Terraform modules from architecture documentation
- [ ] Update all references to Terraform in docs

### Enhance Helm Integration
- [ ] Expand Helm provider to handle infrastructure charts
- [ ] Add support for installing ingress controllers, monitoring via Helm
- [ ] Update service configuration to use Helm for everything post-cluster

### Update Documentation
- [ ] Revise architecture diagrams to show k3d → Helm flow
- [ ] Update integration guide to remove Terraform sections
- [ ] Update examples to show Helm-only approach

## Simplified Workflow

### Before (Complex)
```
devenv up →
  k3d cluster create →
  terraform apply (infrastructure) →
  helm install (applications) →
  services running
```

### After (Simple)  
```
devenv up →
  k3d cluster create →
  helm install (infrastructure + applications) →
  services running
```

### Configuration Example
```yaml
# .devenv/config.yml - No Terraform needed!
services:
  # Infrastructure via Helm
  - name: nginx-ingress
    source:
      type: helm
    chart:
      repository: "https://kubernetes.github.io/ingress-nginx"
      name: "ingress-nginx"
      
  # Application services via Helm  
  - name: user-service
    source:
      type: local
      path: "../user-service"
    chart: "microservice"
```

## Benefits Realized

1. **Faster Onboarding**: Developers only need to understand k3d + Helm
2. **Faster Workflows**: Direct Helm upgrades for hot-reload
3. **Less Configuration**: No Terraform state files or modules to manage
4. **Better Kubernetes Integration**: Helm templates vs Terraform k8s resources
5. **Simpler Debugging**: Fewer tools in the chain when things go wrong

## Future Considerations

If teams need actual cloud resource provisioning (e.g., connecting to real AWS RDS), that can be handled as:
- Separate Terraform modules that output connection details
- External dependency management outside of devenv core
- Optional add-on functionality, not core workflow

---

*This decision exemplifies our "minimize complexity" and "prefer established tools" principles by eliminating an unnecessary abstraction layer.*