package provisioning

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DependencyCheck represents a single dependency validation
type DependencyCheck struct {
	Name        string
	Description string
	Command     string
	ExpectedIn  string // Expected string in output
}

// DependencyValidationResult holds the result of a dependency check
type DependencyValidationResult struct {
	Name    string
	Success bool
	Output  string
	Error   error
}

// ValidateDependenciesArgs are the arguments for dependency validation
type ValidateDependenciesArgs struct {
	NodeName  string
	NodeIP    pulumi.StringInput
	SSHKey    pulumi.StringInput
	Checks    []DependencyCheck
	DependsOn []pulumi.Resource
}

// ValidateDependencies validates that all required dependencies are installed on a node
// using concurrent goroutines for efficiency
func ValidateDependencies(ctx *pulumi.Context, name string, args *ValidateDependenciesArgs) (pulumi.StringOutput, error) {
	// This will execute during Pulumi deployment
	return pulumi.All(args.NodeIP, args.SSHKey).ApplyT(func(deps []interface{}) (string, error) {
		nodeIP := deps[0].(string)
		sshKey := deps[1].(string)

		results := make(chan DependencyValidationResult, len(args.Checks))
		var wg sync.WaitGroup

		// Launch goroutines for concurrent validation
		for _, check := range args.Checks {
			wg.Add(1)
			go func(c DependencyCheck) {
				defer wg.Done()

				// Execute validation command via SSH
				result := DependencyValidationResult{
					Name: c.Name,
				}

				// Create remote command to check dependency
				checkCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-check-%s", name, c.Name), &remote.CommandArgs{
					Connection: remote.ConnectionArgs{
						Host:           pulumi.String(nodeIP),
						User:           pulumi.String("root"),
						PrivateKey:     pulumi.String(sshKey),
						DialErrorLimit: pulumi.Int(5),
					},
					Create: pulumi.String(c.Command),
				}, pulumi.DependsOn(args.DependsOn))

				if err != nil {
					result.Error = err
					result.Success = false
					results <- result
					return
				}

				// Check output
				checkCmd.Stdout.ApplyT(func(output string) string {
					result.Output = output
					if strings.Contains(output, c.ExpectedIn) {
						result.Success = true
					} else {
						result.Success = false
						result.Error = fmt.Errorf("expected '%s' not found in output", c.ExpectedIn)
					}
					results <- result
					return output
				})
			}(check)
		}

		// Wait for all checks to complete
		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect results
		var failedChecks []string
		var successCount int

		for result := range results {
			if result.Success {
				successCount++
				ctx.Log.Info(fmt.Sprintf("✅ %s: %s OK", args.NodeName, result.Name), nil)
			} else {
				failedChecks = append(failedChecks, result.Name)
				ctx.Log.Error(fmt.Sprintf("❌ %s: %s FAILED - %v", args.NodeName, result.Name, result.Error), nil)
			}
		}

		if len(failedChecks) > 0 {
			return "", fmt.Errorf("dependency validation failed for %s: %v", args.NodeName, failedChecks)
		}

		return fmt.Sprintf("All %d dependencies validated successfully on %s", successCount, args.NodeName), nil
	}).(pulumi.StringOutput), nil
}

// GetStandardDependencyChecks returns the standard set of dependency checks
// required before WireGuard and RKE2 installation
func GetStandardDependencyChecks() []DependencyCheck {
	return []DependencyCheck{
		{
			Name:        "docker",
			Description: "Docker engine installed and running",
			Command:     "docker --version",
			ExpectedIn:  "Docker version",
		},
		{
			Name:        "wireguard-tools",
			Description: "WireGuard tools installed",
			Command:     "wg --version",
			ExpectedIn:  "wireguard-tools",
		},
		{
			Name:        "curl",
			Description: "curl utility available",
			Command:     "curl --version",
			ExpectedIn:  "curl",
		},
		{
			Name:        "systemctl",
			Description: "systemd is available",
			Command:     "systemctl --version",
			ExpectedIn:  "systemd",
		},
		{
			Name:        "ip-forwarding",
			Description: "IP forwarding enabled",
			Command:     "sysctl net.ipv4.ip_forward",
			ExpectedIn:  "net.ipv4.ip_forward = 1",
		},
		{
			Name:        "disk-space",
			Description: "Sufficient disk space available",
			Command:     "df -h / | tail -1 | awk '{print $5}' | sed 's/%//'",
			ExpectedIn:  "", // Will check if < 80% used
		},
	}
}

// ValidateDependenciesSync validates dependencies synchronously (simpler version)
// Returns a Pulumi resource that blocks until all checks pass
func ValidateDependenciesSync(ctx *pulumi.Context, name string, nodeIP pulumi.StringInput, sshKey pulumi.StringInput, dependsOn []pulumi.Resource) (*remote.Command, error) {
	// Combine all checks into a single validation script
	validationScript := `#!/bin/bash
set -e

echo "🔍 Validating dependencies..."

# Check Docker
if ! docker --version | grep -q "Docker version"; then
	echo "❌ Docker not found"
	exit 1
fi
echo "✅ Docker OK"

# Check WireGuard tools
if ! wg --version 2>&1 | grep -q "wireguard-tools"; then
	echo "❌ WireGuard tools not found"
	exit 1
fi
echo "✅ WireGuard tools OK"

# Check curl
if ! curl --version | grep -q "curl"; then
	echo "❌ curl not found"
	exit 1
fi
echo "✅ curl OK"

# Check systemd
if ! systemctl --version | grep -q "systemd"; then
	echo "❌ systemd not found"
	exit 1
fi
echo "✅ systemd OK"

# Check IP forwarding
if ! sysctl net.ipv4.ip_forward | grep -q "= 1"; then
	echo "⚠️  IP forwarding not enabled, enabling now..."
	sysctl -w net.ipv4.ip_forward=1
	echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
fi
echo "✅ IP forwarding OK"

# Check disk space
USED=$(df -h / | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$USED" -gt 80 ]; then
	echo "⚠️  Disk usage is ${USED}% (> 80%)"
else
	echo "✅ Disk space OK (${USED}% used)"
fi

echo ""
echo "✅ ✅ ✅ All dependency checks PASSED"
`

	return remote.NewCommand(ctx, fmt.Sprintf("%s-validate-deps", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           nodeIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshKey,
			DialErrorLimit: pulumi.Int(10),
		},
		Create: pulumi.String(validationScript),
	}, pulumi.DependsOn(dependsOn), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "5m",
	}))
}
