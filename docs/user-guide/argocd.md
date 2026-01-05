---
title: ArgoCD GitOps
description: GitOps continuous delivery with ArgoCD integration
sidebar_position: 9
---

# ArgoCD GitOps

sloth-kubernetes provides built-in ArgoCD integration for declarative GitOps continuous delivery, enabling automatic synchronization of your applications from Git repositories.

## Overview

The ArgoCD integration provides:
- **Automated installation**: Install ArgoCD with a single command
- **GitOps repository setup**: Configure your Git repository as the source of truth
- **App of Apps pattern**: Manage multiple applications declaratively
- **Application sync**: Trigger synchronization from the CLI
- **Status monitoring**: Check ArgoCD and application health

## Commands

### argocd install

Install ArgoCD on your Kubernetes cluster.

```bash
# Install ArgoCD with defaults
sloth-kubernetes argocd install --config cluster.lisp

# Install with GitOps repository
sloth-kubernetes argocd install --config cluster.lisp \
  --repo https://github.com/myorg/gitops.git \
  --branch main \
  --apps-path argocd/apps

# Install with App of Apps pattern
sloth-kubernetes argocd install --config cluster.lisp \
  --repo https://github.com/myorg/gitops.git \
  --app-of-apps \
  --app-of-apps-name root-app

# Install specific version
sloth-kubernetes argocd install --config cluster.lisp --version v2.9.3
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace` | ArgoCD namespace | `argocd` |
| `--version` | ArgoCD version (stable, v2.9.3, etc) | `stable` |
| `--repo` | GitOps repository URL | - |
| `--branch` | GitOps repository branch | `main` |
| `--apps-path` | Path to applications in GitOps repo | `argocd/apps` |
| `--app-of-apps` | Enable App of Apps pattern | `false` |
| `--app-of-apps-name` | Name for App of Apps | `root-app` |

**What happens during installation:**
1. Creates the ArgoCD namespace
2. Installs ArgoCD from official manifests
3. Waits for all pods to be ready
4. Optionally configures GitOps repository
5. Sets up App of Apps pattern if enabled

### argocd status

Check the status of ArgoCD installation and applications.

```bash
sloth-kubernetes argocd status --config cluster.lisp
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                     ARGOCD STATUS
═══════════════════════════════════════════════════════════════

Namespace: argocd

Status: Healthy

Pods:
  [OK] argocd-application-controller-0: Running (1/1)
  [OK] argocd-dex-server-6bf8f8b7d5-x2k9m: Running (1/1)
  [OK] argocd-redis-58d7b48b45-7h9ns: Running (1/1)
  [OK] argocd-repo-server-5c8d7b6f4-k3j8n: Running (1/1)
  [OK] argocd-server-7d9f8c6b5-m4k2p: Running (1/1)

Applications: 12 total, 10 synced, 2 out-of-sync
```

### argocd password

Retrieve the ArgoCD admin password.

```bash
sloth-kubernetes argocd password --config cluster.lisp
```

**Example output:**

```
ArgoCD Admin Credentials
========================
Username: admin
Password: aB3cD4eF5gH6iJ7k
```

### argocd apps

List all ArgoCD applications and their sync status.

```bash
sloth-kubernetes argocd apps --config cluster.lisp
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                   ARGOCD APPLICATIONS
═══════════════════════════════════════════════════════════════

NAME                           SYNC STATUS     HEALTH          REPO
--------------------------------------------------------------------------------
cert-manager                   [OK] Synced     [OK] Healthy    https://github.c...
nginx-ingress                  [OK] Synced     [OK] Healthy    https://github.c...
monitoring-stack               [WARN] OutOfSync [OK] Healthy   https://github.c...
database                       [OK] Synced     [OK] Healthy    https://github.c...
api-gateway                    [OK] Synced     [WARN] Degraded https://github.c...
```

### argocd sync

Trigger synchronization of ArgoCD applications.

```bash
# Sync a specific application
sloth-kubernetes argocd sync my-app --config cluster.lisp

# Sync all applications
sloth-kubernetes argocd sync --all --config cluster.lisp
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace` | ArgoCD namespace | `argocd` |
| `--all` | Sync all applications | `false` |

---

## App of Apps Pattern

The App of Apps pattern allows you to manage multiple applications declaratively from a single root application.

### Setting Up App of Apps

```bash
sloth-kubernetes argocd install --config cluster.lisp \
  --repo https://github.com/myorg/gitops.git \
  --app-of-apps \
  --app-of-apps-name cluster-apps
```

### GitOps Repository Structure

```
gitops-repo/
├── argocd/
│   └── apps/
│       ├── cert-manager.yaml
│       ├── nginx-ingress.yaml
│       ├── monitoring.yaml
│       └── database.yaml
├── apps/
│   ├── cert-manager/
│   │   ├── kustomization.yaml
│   │   └── values.yaml
│   ├── nginx-ingress/
│   │   └── values.yaml
│   └── monitoring/
│       └── values.yaml
└── README.md
```

### Example Application Manifest

`argocd/apps/cert-manager.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cert-manager
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://charts.jetstack.io
    targetRevision: v1.13.0
    chart: cert-manager
    helm:
      values: |
        installCRDs: true
  destination:
    server: https://kubernetes.default.svc
    namespace: cert-manager
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

---

## Configuration via LISP

Enable ArgoCD in your cluster configuration:

```lisp
(cluster
  (metadata
    (name "production"))

  (addons
    (argocd
      (enabled true)
      (namespace "argocd")
      (version "stable")
      (gitops-repo "https://github.com/myorg/gitops.git")
      (gitops-branch "main")
      (apps-path "argocd/apps"))))
```

---

## Accessing ArgoCD UI

After installation, access the ArgoCD web interface:

```bash
# Port-forward to local machine
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Open in browser
open https://localhost:8080
```

**Login credentials:**
- Username: `admin`
- Password: Run `sloth-kubernetes argocd password --config cluster.lisp`

---

## Examples

### Complete GitOps Setup

```bash
# 1. Install ArgoCD with GitOps repo
sloth-kubernetes argocd install --config cluster.lisp \
  --repo https://github.com/myorg/k8s-apps.git \
  --branch main \
  --apps-path environments/production

# 2. Check installation status
sloth-kubernetes argocd status --config cluster.lisp

# 3. Get admin password
sloth-kubernetes argocd password --config cluster.lisp

# 4. List deployed applications
sloth-kubernetes argocd apps --config cluster.lisp
```

### Managing Applications

```bash
# Check out-of-sync applications
sloth-kubernetes argocd apps --config cluster.lisp | grep OutOfSync

# Sync specific application
sloth-kubernetes argocd sync monitoring-stack --config cluster.lisp

# Sync all applications after git push
sloth-kubernetes argocd sync --all --config cluster.lisp
```

### CI/CD Integration

```bash
#!/bin/bash
# deploy.sh - Called after git push to trigger sync

# Sync all applications
sloth-kubernetes argocd sync --all --config cluster.lisp

# Wait and check status
sleep 30
sloth-kubernetes argocd apps --config cluster.lisp

# Verify no out-of-sync applications
if sloth-kubernetes argocd apps --config cluster.lisp | grep -q "OutOfSync"; then
  echo "Some applications are out of sync!"
  exit 1
fi

echo "All applications synced successfully!"
```

---

## Troubleshooting

### ArgoCD Pods Not Starting

Check pod status and events:

```bash
sloth-kubernetes kubectl get pods -n argocd
sloth-kubernetes kubectl describe pod <pod-name> -n argocd
```

### Application Stuck in OutOfSync

Check application events and logs:

```bash
# View application details in ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Or check via CLI
sloth-kubernetes kubectl get application <app-name> -n argocd -o yaml
```

### Cannot Reach Git Repository

Verify network connectivity and credentials:

```bash
# Check repo server logs
sloth-kubernetes kubectl logs -n argocd deployment/argocd-repo-server

# Verify repository is configured
sloth-kubernetes kubectl get secret -n argocd -l argocd.argoproj.io/secret-type=repository
```

### Health Check Failed

If the ArgoCD status shows unhealthy:

```bash
# Get detailed pod status
sloth-kubernetes kubectl get pods -n argocd -o wide

# Check specific component logs
sloth-kubernetes kubectl logs -n argocd deployment/argocd-application-controller
sloth-kubernetes kubectl logs -n argocd deployment/argocd-server
```
