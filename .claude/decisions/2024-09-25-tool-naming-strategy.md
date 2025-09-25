# Decision Record: Tool Naming Strategy

**Date**: 2024-09-25
**Status**: Proposed  
**Context**: Choosing tool name based on scope and intended usage

## Problem Statement

The tool is currently named `devenv` but is being developed specifically for Minitab Solution Center (Platform) development. We need to decide whether to use:
1. Domain-specific naming (`plat`, `msc`)
2. Generic naming (`devenv`, `k3dev`)

### Context Factors
- Not planning to open source
- Other teams within MSC/Platform ecosystem may adopt, but not initial goal
- Non-MSC Minitab teams are outside this ecosystem and wouldn't use this tool
- Tool handles k3d + Helm development with built artifacts vs local overrides  
- Potential CI integration for automated testing

## Decision

**Chosen**: `plat` (Platform)

**Rationale**:
1. **Internal Tool Clarity**: Since not open sourcing, domain context adds value rather than limiting scope
2. **Team Communication**: `plat up` immediately communicates Platform development context
3. **MSC Ecosystem Focused**: Works for local dev, CI integration, and potential adoption by other MSC/Platform teams
4. **Memorable**: Short, easy to type, intuitive for team members
5. **Future-Proof**: Platform concept broader than specific product names

## Implementation Changes

### Rename Throughout Codebase
- [ ] Update binary name from `devenv` to `plat`
- [ ] Update CLI command descriptions and help text
- [ ] Update configuration file references (`.plat/config.yml`)
- [ ] Update all documentation and examples
- [ ] Update README and architecture docs

### Domain-Specific Optimizations
Since we're domain-specific, we can make opinionated choices:
- [ ] Default registry patterns for Minitab infrastructure
- [ ] Built-in Platform service templates  
- [ ] Standard ingress configurations
- [ ] Pre-configured monitoring/logging for Platform patterns

## Usage Examples

### Local Development
```bash
plat init --template platform-microservices
plat up
plat logs --service user-api
plat restart payment-service
```

### CI Integration  
```bash
plat up --env integration-test --wait-for-ready
plat test --services user-api,payment-api
plat cleanup
```

### Configuration
```yaml
# .plat/config.yml - Platform-specific defaults
apiVersion: plat/v1
kind: Environment
metadata:
  name: platform-local
spec:
  defaults:
    registry: "minitab.registry.local"
    domain: "platform.local"
```

## Rejected Alternatives

**`msc` (Minitab Solution Center)**:
- Too cryptic for new team members
- Tied to specific product vs broader platform concept
- Less intuitive in CI contexts

**`devenv` (Generic)**:
- Good for open source, but we're not open sourcing
- Misses opportunity for domain-specific optimizations
- Less clear communication of purpose

**`k3dev`, `k8dev` (Kubernetes-focused)**:
- Too generic given our specific use case
- Doesn't communicate Platform context
- Would require more configuration vs conventions

## Benefits Realized

1. **Clear Purpose**: Anyone using `plat` knows it's for Platform development
2. **Convention Over Configuration**: Can build in Platform team patterns
3. **MSC Ecosystem Adoption**: Other MSC/Platform teams can understand scope and potentially adopt  
4. **CI/Local Consistency**: Same tool, same commands across environments
5. **Future Flexibility**: Platform concept accommodates evolution

---

*This naming choice enables domain-specific optimizations while maintaining the simplicity goals of our north star.*