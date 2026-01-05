---
title: Backup & Restore
description: Manage Kubernetes cluster backups with Velero integration
sidebar_position: 8
---

# Backup & Restore

sloth-kubernetes provides comprehensive backup and restore capabilities using Velero, enabling you to protect your Kubernetes applications and data.

## Overview

The backup system provides:
- **Full cluster backups**: Back up all namespaces and resources
- **Selective backups**: Target specific namespaces or resource types
- **Scheduled backups**: Automated backups with cron expressions
- **Volume snapshots**: PersistentVolume snapshot support
- **Point-in-time recovery**: Restore from any backup
- **Multi-cloud storage**: Support for AWS S3, GCP, Azure, MinIO

## Prerequisites

Velero must be installed in your cluster before using backup operations:

```bash
# Check if Velero is installed
sloth-kubernetes backup status

# Install Velero with your storage provider
sloth-kubernetes backup install \
  --provider aws \
  --bucket my-backup-bucket \
  --region us-east-1 \
  --secret-file ./credentials
```

---

## Commands

### backup install

Install Velero in the cluster with the specified storage provider.

```bash
sloth-kubernetes backup install \
  --provider aws \
  --bucket my-velero-bucket \
  --region us-east-1 \
  --secret-file ./aws-credentials
```

**Supported providers:**
- `aws` - Amazon S3
- `gcp` - Google Cloud Storage
- `azure` - Azure Blob Storage
- `minio` - MinIO (S3-compatible)

**Flags:**

| Flag | Description | Required |
|------|-------------|----------|
| `--provider` | Storage provider (aws, gcp, azure, minio) | Yes |
| `--bucket` | Storage bucket name | Yes |
| `--region` | Storage region | No |
| `--secret-file` | Path to credentials file | Yes |

### backup status

Check if Velero is installed and show its status.

```bash
sloth-kubernetes backup status
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                     VELERO STATUS
═══════════════════════════════════════════════════════════════

[OK] Velero is installed

Backup Locations: 1 configured
  - default (aws)

Backups: 15 total
Schedules: 2 configured
```

### backup create

Create a new backup of the cluster or specific namespaces.

```bash
# Create a full cluster backup
sloth-kubernetes backup create my-backup

# Backup specific namespaces
sloth-kubernetes backup create my-backup --namespaces default,app,database

# Exclude certain namespaces
sloth-kubernetes backup create my-backup --exclude-namespaces kube-system,monitoring

# Backup specific resources only
sloth-kubernetes backup create my-backup --resources deployments,services,configmaps

# Set custom retention period (default: 30 days)
sloth-kubernetes backup create my-backup --ttl 168h  # 7 days

# Backup without volume snapshots
sloth-kubernetes backup create my-backup --snapshot-volumes=false

# Wait for backup to complete
sloth-kubernetes backup create my-backup --wait

# Dry-run to see what would be backed up
sloth-kubernetes backup create my-backup --dry-run
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespaces` | Namespaces to include (comma-separated) | All |
| `--exclude-namespaces` | Namespaces to exclude | - |
| `--resources` | Resources to include | All |
| `--exclude-resources` | Resources to exclude | - |
| `--ttl` | Backup retention period | `720h` (30 days) |
| `--storage-location` | Backup storage location name | Default |
| `--labels` | Labels to apply (key=value) | - |
| `--snapshot-volumes` | Take PV snapshots | `true` |
| `--wait` | Wait for backup to complete | `false` |
| `--timeout` | Timeout when using --wait | `30m` |
| `--dry-run` | Preview without creating | `false` |

### backup list

List all backups in the cluster.

```bash
sloth-kubernetes backup list

# Output as JSON
sloth-kubernetes backup list --json
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                         BACKUPS
═══════════════════════════════════════════════════════════════

NAME                           STATUS          ERRORS     WARNINGS   CREATED
------------------------------------------------------------------------------------------
daily-backup-20240115          Completed       0          0          2024-01-15 00:00:05
daily-backup-20240114          Completed       0          2          2024-01-14 00:00:03
manual-backup-pre-upgrade      Completed       0          0          2024-01-13 14:30:22
weekly-backup-20240107         Completed       0          0          2024-01-07 00:00:08

Total: 4 backups
```

### backup describe

Show detailed information about a specific backup.

```bash
sloth-kubernetes backup describe my-backup
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                     BACKUP DETAILS
═══════════════════════════════════════════════════════════════

  Name:       my-backup
  Status:     Completed
  Phase:      Completed
  Namespaces: default, app, database
  Started:    2024-01-15 10:30:00
  Completed:  2024-01-15 10:35:22
  Duration:   5m22s
  Expires:    2024-02-14 10:30:00
  Errors:     0
  Warnings:   0
  Location:   default
```

### backup delete

Delete a backup.

```bash
sloth-kubernetes backup delete my-backup

# Dry-run to see what would be deleted
sloth-kubernetes backup delete my-backup --dry-run
```

### backup restore

Restore from an existing backup.

```bash
# Restore entire backup
sloth-kubernetes backup restore --from-backup my-backup

# Restore specific namespaces
sloth-kubernetes backup restore --from-backup my-backup --namespaces app,database

# Restore without persistent volumes
sloth-kubernetes backup restore --from-backup my-backup --restore-volumes=false

# Preserve NodePort values
sloth-kubernetes backup restore --from-backup my-backup --preserve-nodeports

# Wait for restore to complete
sloth-kubernetes backup restore --from-backup my-backup --wait

# Dry-run to see what would be restored
sloth-kubernetes backup restore --from-backup my-backup --dry-run
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--from-backup` | Backup name to restore from | Required |
| `--namespaces` | Namespaces to restore | All from backup |
| `--exclude-namespaces` | Namespaces to exclude | - |
| `--resources` | Resources to restore | All |
| `--exclude-resources` | Resources to exclude | - |
| `--restore-volumes` | Restore persistent volumes | `true` |
| `--preserve-nodeports` | Preserve original NodePort values | `false` |
| `--wait` | Wait for restore to complete | `false` |
| `--timeout` | Timeout when using --wait | `30m` |
| `--dry-run` | Preview without restoring | `false` |

### backup restore-list

List all restore operations.

```bash
sloth-kubernetes backup restore-list
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                         RESTORES
═══════════════════════════════════════════════════════════════

NAME                           BACKUP                    STATUS          ERRORS     WARNINGS
-----------------------------------------------------------------------------------------------
restore-20240115-103045        my-backup                 Completed       0          0
restore-20240112-154522        daily-backup-20240112     Completed       0          1

Total: 2 restores
```

### backup locations

List all configured backup storage locations.

```bash
sloth-kubernetes backup locations
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                 BACKUP STORAGE LOCATIONS
═══════════════════════════════════════════════════════════════

NAME                 PROVIDER        BUCKET                    REGION          DEFAULT
------------------------------------------------------------------------------------------
default              aws             my-velero-bucket          us-east-1       Yes
secondary            gcp             backup-bucket-gcp         us-central1     No
```

---

## Scheduled Backups

### schedule create

Create automated backup schedules using cron expressions.

```bash
# Daily backup at midnight
sloth-kubernetes backup schedule create daily-backup --schedule "0 0 * * *"

# Every 6 hours
sloth-kubernetes backup schedule create frequent-backup --schedule "0 */6 * * *"

# Weekly on Sunday at midnight
sloth-kubernetes backup schedule create weekly-backup --schedule "0 0 * * 0"

# Monthly on the 1st at midnight
sloth-kubernetes backup schedule create monthly-backup --schedule "0 0 1 * *"

# Schedule with specific namespaces
sloth-kubernetes backup schedule create app-backup \
  --schedule "0 0 * * *" \
  --namespaces app,database

# Schedule with custom retention
sloth-kubernetes backup schedule create short-retention \
  --schedule "0 */12 * * *" \
  --ttl 48h

# Dry-run to preview
sloth-kubernetes backup schedule create daily-backup \
  --schedule "0 0 * * *" \
  --dry-run
```

**Cron expression format:** `minute hour day-of-month month day-of-week`

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--schedule` | Cron expression | Required |
| `--namespaces` | Namespaces to include | All |
| `--exclude-namespaces` | Namespaces to exclude | - |
| `--ttl` | Backup retention period | `720h` |
| `--snapshot-volumes` | Take PV snapshots | `true` |
| `--dry-run` | Preview without creating | `false` |

### schedule list

List all backup schedules.

```bash
sloth-kubernetes backup schedule list
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
                     BACKUP SCHEDULES
═══════════════════════════════════════════════════════════════

NAME                      SCHEDULE             PAUSED     LAST BACKUP
-------------------------------------------------------------------------------------
daily-backup              0 0 * * *            No         2024-01-15 00:00:05
weekly-backup             0 0 * * 0            No         2024-01-14 00:00:03
app-hourly                0 * * * *            Yes        2024-01-10 15:00:02

Total: 3 schedules
```

### schedule delete

Delete a backup schedule.

```bash
sloth-kubernetes backup schedule delete daily-backup

# Dry-run
sloth-kubernetes backup schedule delete daily-backup --dry-run
```

### schedule pause

Pause a backup schedule.

```bash
sloth-kubernetes backup schedule pause daily-backup
```

### schedule unpause

Resume a paused backup schedule.

```bash
sloth-kubernetes backup schedule unpause daily-backup
```

---

## Global Flags

These flags apply to all backup commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--kubeconfig` | Path to kubeconfig file | `~/.kube/config` |
| `--velero-namespace` | Velero namespace | `velero` |
| `--json` | Output in JSON format | `false` |

---

## Examples

### Disaster Recovery Setup

Set up comprehensive disaster recovery:

```bash
# Install Velero with AWS S3
sloth-kubernetes backup install \
  --provider aws \
  --bucket dr-backups \
  --region us-west-2 \
  --secret-file ./aws-creds

# Create daily full cluster backup
sloth-kubernetes backup schedule create daily-full \
  --schedule "0 2 * * *" \
  --ttl 720h

# Create hourly app backup
sloth-kubernetes backup schedule create hourly-apps \
  --schedule "0 * * * *" \
  --namespaces app,api,frontend \
  --ttl 72h

# Create weekly long-term backup
sloth-kubernetes backup schedule create weekly-archive \
  --schedule "0 3 * * 0" \
  --ttl 8760h  # 1 year
```

### Pre-Upgrade Backup

Create a backup before cluster upgrade:

```bash
# Create backup with descriptive name
sloth-kubernetes backup create pre-upgrade-v128-to-v129 \
  --wait \
  --labels env=production,reason=upgrade

# Verify backup completed
sloth-kubernetes backup describe pre-upgrade-v128-to-v129

# Proceed with upgrade
sloth-kubernetes upgrade apply --to v1.29.0
```

### Application Migration

Migrate applications between clusters:

```bash
# Source cluster: Create application backup
sloth-kubernetes backup create app-migration \
  --namespaces myapp \
  --wait

# Target cluster: Restore application
sloth-kubernetes backup restore \
  --from-backup app-migration \
  --wait
```

### Namespace Recovery

Restore a specific namespace after accidental deletion:

```bash
# Find backup containing the namespace
sloth-kubernetes backup list

# Describe backup to verify contents
sloth-kubernetes backup describe daily-backup-20240115

# Restore only the deleted namespace
sloth-kubernetes backup restore \
  --from-backup daily-backup-20240115 \
  --namespaces deleted-namespace \
  --wait
```

### Selective Resource Restore

Restore specific resources without overwriting existing:

```bash
# Restore only ConfigMaps and Secrets
sloth-kubernetes backup restore \
  --from-backup my-backup \
  --resources configmaps,secrets \
  --namespaces app

# Restore everything except Deployments (keep current versions running)
sloth-kubernetes backup restore \
  --from-backup my-backup \
  --exclude-resources deployments \
  --namespaces app
```

---

## Credential Files

### AWS S3

Create `aws-credentials` file:

```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

### Google Cloud Storage

Create `gcp-credentials.json` using a service account key.

### Azure Blob Storage

Create `azure-credentials` file:

```
AZURE_SUBSCRIPTION_ID=your-subscription-id
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
AZURE_RESOURCE_GROUP=your-resource-group
AZURE_CLOUD_NAME=AzurePublicCloud
```

### MinIO

Create `minio-credentials` file (same format as AWS):

```ini
[default]
aws_access_key_id = MINIO_ACCESS_KEY
aws_secret_access_key = MINIO_SECRET_KEY
```

---

## Troubleshooting

### Velero Not Installed

```
Error: velero is not installed. Run 'sloth-kubernetes backup install' first
```

Install Velero with your storage provider:

```bash
sloth-kubernetes backup install --provider aws --bucket my-bucket --secret-file ./creds
```

### Backup Stuck in InProgress

Check Velero pods:

```bash
sloth-kubernetes kubectl get pods -n velero
sloth-kubernetes kubectl logs -n velero deployment/velero
```

### Restore Fails with Conflicts

If resources already exist:

```bash
# Restore with exclude to avoid conflicts
sloth-kubernetes backup restore \
  --from-backup my-backup \
  --exclude-resources persistentvolumeclaims
```

### Volume Snapshots Not Working

Ensure your storage provider supports CSI snapshots:

```bash
# Check VolumeSnapshotClass exists
sloth-kubernetes kubectl get volumesnapshotclass
```

### Backup Taking Too Long

For large clusters, exclude non-essential namespaces:

```bash
sloth-kubernetes backup create quick-backup \
  --exclude-namespaces monitoring,logging,kube-system \
  --timeout 60m
```
