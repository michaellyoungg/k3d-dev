# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This project is developing a unified command-line tool (`plat`) for managing local microservice development environments. The tool integrates k3d (lightweight Kubernetes) and Helm (package management) to provide a seamless developer experience.

**Core Goal**: Simplify and standardize local microservice development workflows by abstracting away infrastructure complexity while maintaining flexibility and power.

## Project Structure

```
.
├── .claude/                # Project documentation and planning
│   ├── CLAUDE.md          # This file
│   └── docs/              # Comprehensive project documentation
│       ├── 00-engineering-principles.md # Engineering partnership principles
│       ├── 01-project-overview.md    # Vision and problem statement
│       ├── 02-requirements.md        # Functional and non-functional requirements  
│       ├── 03-use-cases.md          # User scenarios and personas
│       ├── 04-ergonomics.md         # CLI design and UX patterns
│       ├── 05-architecture.md       # System architecture and components
│       ├── 06-technology-integration.md # k3d, Helm, Terraform integration
│       └── 07-decision-workflow.md   # Decision-making templates and process
└── (implementation to be added)
```

## Key Concepts

**Target Technologies**:
- **k3d**: Lightweight Kubernetes clusters for local development
- **Helm**: Application packaging and deployment

**Primary Use Cases**:
- First-time project setup and environment initialization
- Daily development workflows with hot-reload and debugging
- Multi-service integration testing
- Debugging production issues locally
- Team collaboration on shared development environments

## Current Status

**Phase**: Planning and Documentation
- ✅ Project vision and requirements defined
- ✅ Use cases and user personas documented
- ✅ CLI ergonomics and command structure planned
- ✅ System architecture designed
- ✅ Technology integration patterns specified
- 🚧 Implementation planning in progress

## Development Approach

When implementing this tool:

1. **Start Small**: Begin with core environment lifecycle (init, up, stop, status)
2. **Prioritize Ergonomics**: Focus on developer experience and simple commands
3. **Embrace Standards**: Follow established CLI conventions and configuration patterns
4. **Design for Extension**: Plugin architecture for custom service types and workflows
5. **Validate Early**: Test with real microservice projects and gather feedback

## Architecture Highlights

- **Layered Configuration**: Global → Project → Environment → Runtime overrides
- **Service Orchestration**: Dependency management with health checks
- **Plugin System**: Extensible service types and custom commands
- **State Management**: Local SQLite for environment state persistence
- **Tool Integration**: Direct API/CLI integration with k3d and Helm

## Documentation

All project planning and architectural decisions are captured in `./.claude/docs/`. This comprehensive documentation should be consulted when making implementation decisions or understanding the project's scope and vision.