#!/bin/bash
set -e

echo "Setting up Pulumi with MinIO backend..."

# Set AWS credentials for MinIO
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY='Pulumi@24081995'
export AWS_REGION=us-east-1
export PULUMI_BACKEND_URL='s3://pulumi-state?endpoint=s3.lady-guica.chalkan3.com.br&disableSSL=false'

# Try to login to Pulumi
echo "Logging into Pulumi S3 backend..."
pulumi login "$PULUMI_BACKEND_URL" --non-interactive || {
    echo "Failed to login to Pulumi backend"
    echo "Trying with force_path_style..."
    export AWS_S3_FORCE_PATH_STYLE=true
    pulumi login "$PULUMI_BACKEND_URL" --non-interactive || {
        echo "Failed with path style. Using local backend instead..."
        pulumi login --local --non-interactive
    }
}

# Initialize the stack
STACK_NAME="dev"
echo "Selecting or creating stack: $STACK_NAME"
pulumi stack select "$STACK_NAME" 2>/dev/null || pulumi stack init "$STACK_NAME" --non-interactive

# Show current configuration
echo ""
echo "Current Pulumi configuration:"
pulumi whoami
pulumi stack ls

echo ""
echo "Setup complete! You can now run:"
echo "  pulumi preview"
echo "  pulumi up"