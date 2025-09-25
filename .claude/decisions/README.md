# Decision Records

This directory contains all architectural and technical decisions made during the development of the devenv tool. Each decision follows the format outlined in our [decision workflow](./../docs/07-decision-workflow.md).

## Decisions

### 2024-09-24
- [CLI Framework Choice](./2024-09-24-cli-framework-choice.md) - Selected Go with Cobra over Node.js with Commander

### 2024-09-25  
- [Service Configuration Design](./2024-09-25-service-configuration-design.md) - YAML-based flexible service sources (local/registry/helm)
- [Terraform vs Helm for Development Workflows](./2024-09-25-terraform-vs-helm-dev-workflows.md) - Analysis of hot-reload workflow tooling
- [Remove Terraform from Toolchain](./2024-09-25-remove-terraform-simplify-toolchain.md) - Simplify to k3d + Helm only
- [Product Strategy and North Star](./2024-09-25-product-strategy-north-star.md) - Strategic boundaries and user experience principles
- [Tool Naming Strategy](./2024-09-25-tool-naming-strategy.md) - Choose `plat` for MSC/Platform ecosystem focus
- [Configuration File Design](./2024-09-25-configuration-file-design.md) - **Layered config with capability declarations and CLI mode control**

## Decision Status

- **Accepted**: Final decisions that are implemented
- **Proposed**: Decisions under review or waiting for implementation
- **Superseded**: Decisions that have been replaced by newer decisions

## Quick Reference

**Product Vision**: "Docker Compose for Kubernetes" - 5-minute microservices setup
**Architecture**: k3d + Helm (Terraform removed for simplicity)
**Configuration**: Convention over configuration, <20 lines YAML for 80% of use cases
**CLI Framework**: Go with Cobra  
**User Experience**: Progressive disclosure, immediate feedback, smart defaults