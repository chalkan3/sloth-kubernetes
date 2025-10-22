# Login Feature - Credential Management

## Overview

The `login` command provides a secure way to store and manage cloud provider credentials, similar to how Pulumi manages its login state.

## Features

- Interactive credential input with hidden passwords
- Secure storage in `~/.sloth-kubernetes/credentials` with 0600 permissions
- Automatic credential loading for all commands
- Support for multiple cloud providers
- Per-provider configuration
- Credential masking in output

## Usage

### Configure All Providers

```bash
sloth-kubernetes login
```

This will interactively prompt for:
- DigitalOcean API token
- Linode API token

### Configure a Specific Provider

```bash
# Configure only DigitalOcean
sloth-kubernetes login --provider digitalocean

# Configure only Linode
sloth-kubernetes login --provider linode
```

### Overwrite Existing Credentials

```bash
sloth-kubernetes login --overwrite
```

This will skip the confirmation prompt and overwrite all existing credentials.

## How It Works

### 1. Credential Storage

Credentials are stored in `~/.sloth-kubernetes/credentials`:

```bash
# Sloth Kubernetes Cloud Provider Credentials
# This file contains sensitive API tokens - keep it secure!
# File permissions: 0600 (read/write for owner only)

DIGITALOCEAN_TOKEN=dop_v1_xxxxxxxxxxxxx
LINODE_TOKEN=xxxxxxxxxxxxxxxxxx
```

The file is created with restrictive permissions (0600) to ensure only the owner can read/write.

### 2. Automatic Loading

Every command automatically loads credentials from the saved file before execution. This means you can run commands without setting environment variables:

```bash
# No need to set DIGITALOCEAN_TOKEN or LINODE_TOKEN
sloth-kubernetes deploy -c cluster.yaml
```

### 3. Priority Order

Credentials are resolved in this order:
1. Command-line flags (if implemented)
2. Environment variables
3. Saved credentials file

This means saved credentials act as defaults but can be overridden by environment variables if needed.

## Security Considerations

1. **File Permissions**: The credentials file is created with 0600 permissions (read/write for owner only)
2. **Hidden Input**: Tokens are entered with hidden input (terminal password mode)
3. **Masked Display**: Existing tokens are shown masked (e.g., `dop_...xxx`)
4. **No CLI Exposure**: Tokens are never passed as command-line arguments
5. **Home Directory**: Credentials are stored in the user's home directory

## Example Workflow

### First Time Setup

```bash
$ sloth-kubernetes login

🔐 Cloud Provider Login

Enter DigitalOcean API token (hidden): ****
  ✓ DigitalOcean token configured
Enter Linode API token (hidden): ****
  ✓ Linode token configured

✓ Credentials saved successfully!
  Location: /Users/username/.sloth-kubernetes/credentials

Note: You can now deploy clusters without setting environment variables.
```

### Updating Credentials

```bash
$ sloth-kubernetes login

🔐 Cloud Provider Login

DigitalOcean token already configured: dop_...cf1
  Overwrite DigitalOcean token? (y/N): y
Enter DigitalOcean API token (hidden): ****
  ✓ DigitalOcean token configured

Linode token already configured: 1275...f11
  Overwrite Linode token? (y/N): n
  Skipping Linode

✓ Credentials saved successfully!
```

### Using with Deploy

```bash
# No environment variables needed!
$ sloth-kubernetes deploy -c cluster.yaml

🚀 Deploying Kubernetes Cluster
...
```

## Credential File Location

The credentials are stored in:
- **macOS/Linux**: `~/.sloth-kubernetes/credentials`
- **Windows**: `%USERPROFILE%\.sloth-kubernetes\credentials`

## Comparison with Other Tools

### Pulumi
```bash
pulumi login
# Stores state backend configuration
```

### sloth-kubernetes
```bash
sloth-kubernetes login
# Stores cloud provider API tokens
```

### AWS CLI
```bash
aws configure
# Stores credentials in ~/.aws/credentials
```

### Docker
```bash
docker login
# Stores registry credentials in ~/.docker/config.json
```

## Troubleshooting

### Credentials Not Loading

If credentials aren't being loaded:

1. Check file exists:
```bash
ls -la ~/.sloth-kubernetes/credentials
```

2. Check file permissions:
```bash
chmod 600 ~/.sloth-kubernetes/credentials
```

3. Verify file contents:
```bash
cat ~/.sloth-kubernetes/credentials
```

### Override Saved Credentials

Use environment variables to temporarily override:

```bash
export DIGITALOCEAN_TOKEN="different_token"
sloth-kubernetes deploy -c cluster.yaml
```

## Future Enhancements

Potential improvements:
- [ ] Encryption of stored credentials
- [ ] Support for credential rotation
- [ ] Integration with system keychains (macOS Keychain, Windows Credential Manager)
- [ ] Multi-profile support (dev, staging, prod)
- [ ] Credential expiration warnings
- [ ] Integration with cloud provider OAuth flows
