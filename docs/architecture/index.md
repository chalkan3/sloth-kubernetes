---
layout: default
title: Architecture
nav_order: 4
has_children: true
---

# Architecture

Understanding how sloth-kubernetes works under the hood.

## Overview

sloth-kubernetes is designed as a unified orchestration tool that combines:
- Infrastructure provisioning (Pulumi)
- Configuration management (Salt)
- Kubernetes operations (kubectl)
- Package management (Helm/Kustomize)

## Design Principles

1. **Single Binary** - No external dependencies for core functionality
2. **Multi-Cloud Native** - First-class support for multiple providers
3. **Security First** - Bastion architecture and encrypted VPN mesh
4. **Automation** - Minimal manual intervention required
5. **Extensibility** - Plugin architecture for custom providers

## Components

- [Overview](overview) - High-level architecture
- [Orchestration Flow](orchestration) - Deployment phases
- [Networking](networking) - VPN mesh and CNI
- [Security Model](security) - Security architecture
