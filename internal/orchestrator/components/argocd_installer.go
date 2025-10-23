package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// ArgoCDInstallerComponent outputs
type ArgoCDInstallerComponent struct {
	pulumi.ResourceState

	AdminPassword pulumi.StringOutput `pulumi:"adminPassword"`
	Status        pulumi.StringOutput `pulumi:"status"`
}

// NewArgoCDInstallerComponent creates a new ArgoCD installer component
func NewArgoCDInstallerComponent(
	ctx *pulumi.Context,
	name string,
	argoCDConfig *config.ArgoCDConfig,
	nodes []*RealNodeComponent,
	bastionComponent *BastionComponent,
	sshPrivateKey pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*ArgoCDInstallerComponent, error) {
	if argoCDConfig == nil || !argoCDConfig.Enabled {
		return nil, nil // ArgoCD not enabled
	}

	component := &ArgoCDInstallerComponent{}
	err := ctx.RegisterComponentResource("sloth:kubernetes:ArgoCDInstaller", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Find master nodes (first 3 nodes are masters by convention)
	var masters []*RealNodeComponent
	for i, node := range nodes {
		if i < 3 {
			masters = append(masters, node)
		}
	}

	if len(masters) == 0 {
		return nil, fmt.Errorf("no master nodes found for ArgoCD installation")
	}

	// Use first master node
	firstMaster := masters[0]

	// Set defaults
	namespace := argoCDConfig.Namespace
	if namespace == "" {
		namespace = "argocd"
	}

	version := argoCDConfig.Version
	if version == "" {
		version = "stable"
	}

	gitopsBranch := argoCDConfig.GitOpsRepoBranch
	if gitopsBranch == "" {
		gitopsBranch = "main"
	}

	appsPath := argoCDConfig.AppsPath
	if appsPath == "" {
		appsPath = "argocd/apps"
	}

	// Setup connection args
	connArgs := &remote.ConnectionArgs{
		Host:           firstMaster.WireGuardIP, // Use VPN IP for private network
		User:           pulumi.String("root"),
		PrivateKey:     sshPrivateKey,
		DialErrorLimit: pulumi.Int(30),
	}

	if bastionComponent != nil {
		connArgs.Proxy = &remote.ProxyConnectionArgs{
			Host:       bastionComponent.PublicIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		}
	}

	// Step 1: Install ArgoCD
	ctx.Log.Info("🚀 Step 1/3: Installing ArgoCD...", nil)
	installCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-install", name), &remote.CommandArgs{
		Connection: connArgs,
		Create: pulumi.String(fmt.Sprintf(`#!/bin/bash
set -e

echo "════════════════════════════════════════════════════════════"
echo "🚀 Installing ArgoCD GitOps"
echo "════════════════════════════════════════════════════════════"
echo ""

# Create ArgoCD namespace
echo "📦 Creating namespace: %s"
kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -

# Install ArgoCD
echo "📥 Installing ArgoCD version: %s"
kubectl apply -n %s -f https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml

echo ""
echo "⏳ Waiting for ArgoCD pods to be ready (timeout: 300s)..."
kubectl wait --for=condition=Ready pods --all -n %s --timeout=300s || {
  echo "❌ ArgoCD pods not ready in time. Checking pod status:"
  kubectl get pods -n %s
  exit 1
}

echo ""
echo "✅ ArgoCD installed successfully!"
		`, namespace, namespace, version, namespace, version, namespace, namespace)),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create ArgoCD install command: %w", err)
	}

	// Step 2: Clone GitOps repo and apply manifests
	ctx.Log.Info("🚀 Step 2/3: Applying GitOps manifests...", nil)
	var applyCmd *remote.Command
	if argoCDConfig.GitOpsRepoURL != "" {
		applyCmd, err = remote.NewCommand(ctx, fmt.Sprintf("%s-apply-manifests", name), &remote.CommandArgs{
			Connection: connArgs,
			Create: pulumi.String(fmt.Sprintf(`#!/bin/bash
set -e

echo ""
echo "📂 Cloning GitOps repository..."
echo "  Repository: %s"
echo "  Branch: %s"
echo "  Apps Path: %s"
echo ""

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

cd $TEMP_DIR

# Clone the GitOps repository
git clone -b %s %s gitops-repo 2>&1 | grep -v "Cloning into" || true

# Apply manifests from apps path
if [ -d "gitops-repo/%s" ]; then
	echo "📋 Applying manifests from %s..."
	kubectl apply -f gitops-repo/%s/ -n %s --recursive
	echo "✅ Manifests applied successfully!"
else
	echo "⚠️  Warning: Apps path 'gitops-repo/%s' not found"
	echo "    Available directories:"
	ls -la gitops-repo/ || true
fi

echo ""
echo "🎉 GitOps applications deployed!"
		`, argoCDConfig.GitOpsRepoURL, gitopsBranch, appsPath, gitopsBranch, argoCDConfig.GitOpsRepoURL, appsPath, appsPath, appsPath, namespace, appsPath)),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{installCmd}))
		if err != nil {
			return nil, fmt.Errorf("failed to create apply manifests command: %w", err)
		}
	}

	// Step 3: Get ArgoCD admin password
	ctx.Log.Info("🚀 Step 3/3: Retrieving ArgoCD admin password...", nil)
	var passwordDeps []pulumi.Resource
	if applyCmd != nil {
		passwordDeps = []pulumi.Resource{applyCmd}
	} else {
		passwordDeps = []pulumi.Resource{installCmd}
	}

	getPasswordCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-get-password", name), &remote.CommandArgs{
		Connection: connArgs,
		Create: pulumi.String(fmt.Sprintf(`#!/bin/bash
set -e

# Wait a bit for the secret to be created
sleep 5

# Get ArgoCD admin password
kubectl -n %s get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" 2>/dev/null | base64 -d || echo "password-not-ready"
		`, namespace)),
	}, pulumi.Parent(component), pulumi.DependsOn(passwordDeps))
	if err != nil {
		return nil, fmt.Errorf("failed to create get password command: %w", err)
	}

	component.AdminPassword = getPasswordCmd.Stdout
	component.Status = pulumi.String("installed").ToStringOutput()

	// Register outputs
	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"adminPassword": component.AdminPassword,
		"status":        component.Status,
	})

	ctx.Log.Info("✅ ArgoCD installation completed!", nil)

	return component, nil
}
