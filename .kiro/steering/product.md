# Product Overview

Cutepod is an ephemeral, Kubernetes-inspired orchestration tool for local container management using Podman. It bridges the gap between `podman-compose` and Kubernetes in complexity and capability.

## Core Purpose
- Local container orchestration using declarative YAML charts
- Kubernetes-inspired syntax with Go templating (similar to Helm)
- Daemonless and ephemeral - all operations are CLI-based
- Supports containers, secrets, networks, volumes, and custom extensions

## Key Features
- **Declarative Reconciliation**: Automatically applies, upgrades, and reconciles Podman containers
- **Secret Management**: Inline secrets and external secret injection via `CuteSecret` and `CuteSecretStore`
- **Extensibility**: Custom resource transformations through `CuteExtension`
- **Restart Recovery**: `cutepod reinit` handles container restarts after system/Podman restarts

## Target Users
Developers who need local container orchestration that's more structured than docker-compose but simpler than full Kubernetes.