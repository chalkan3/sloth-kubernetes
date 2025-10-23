package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NodeStatus represents the health status of a node
type NodeStatus struct {
	NodeName  string
	IsHealthy bool
	LastCheck time.Time
	Error     error
	Services  map[string]bool
}

// HealthChecker manages health checks for nodes
type HealthChecker struct {
	ctx           *pulumi.Context
	nodes         []*providers.NodeOutput
	sshKeyPath    string
	statuses      map[string]*NodeStatus
	mu            sync.RWMutex
	checkInterval time.Duration
	timeout       time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(ctx *pulumi.Context) *HealthChecker {
	return &HealthChecker{
		ctx:           ctx,
		nodes:         make([]*providers.NodeOutput, 0),
		statuses:      make(map[string]*NodeStatus),
		checkInterval: 10 * time.Second,
		timeout:       5 * time.Minute,
	}
}

// AddNode adds a node to be monitored
func (h *HealthChecker) AddNode(node *providers.NodeOutput) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nodes = append(h.nodes, node)
	h.statuses[node.Name] = &NodeStatus{
		NodeName:  node.Name,
		IsHealthy: false,
		Services:  make(map[string]bool),
	}
}

// SetSSHKeyPath sets the SSH key path for connections
func (h *HealthChecker) SetSSHKeyPath(path string) {
	h.sshKeyPath = path
}

// WaitForNodesReady waits until all nodes are ready with required services
func (h *HealthChecker) WaitForNodesReady(requiredServices []string) error {
	h.ctx.Log.Info("Starting health checks for all nodes", nil)

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Create a channel to signal when all nodes are ready
	readyChan := make(chan bool)
	errorChan := make(chan error)

	// Start health check goroutines for each node
	var wg sync.WaitGroup
	for _, node := range h.nodes {
		wg.Add(1)
		go h.checkNodeHealth(ctx, &wg, node, requiredServices, readyChan, errorChan)
	}

	// Monitor goroutine
	go func() {
		wg.Wait()
		close(readyChan)
	}()

	// Status reporter goroutine
	go h.reportStatus(ctx)

	// Wait for all nodes to be ready or timeout
	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for nodes to be ready")
	case err := <-errorChan:
		return err
	case <-readyChan:
		h.ctx.Log.Info("All nodes are ready!", nil)
		return nil
	}
}

// checkNodeHealth continuously checks the health of a single node
func (h *HealthChecker) checkNodeHealth(ctx context.Context, wg *sync.WaitGroup, node *providers.NodeOutput, requiredServices []string, readyChan chan<- bool, errorChan chan<- error) {
	defer wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthy, err := h.performHealthCheck(node, requiredServices)

			h.mu.Lock()
			status := h.statuses[node.Name]
			status.IsHealthy = healthy
			status.LastCheck = time.Now()
			status.Error = err
			h.mu.Unlock()

			if healthy {
				h.ctx.Log.Info("Node is ready", nil)

				// Check if all nodes are ready
				if h.allNodesReady() {
					select {
					case readyChan <- true:
					default:
					}
					return
				}
			}

			if err != nil && !isRecoverableError(err) {
				errorChan <- fmt.Errorf("node %s health check failed: %w", node.Name, err)
				return
			}
		}
	}
}

// performHealthCheck performs actual health checks on a node
func (h *HealthChecker) performHealthCheck(node *providers.NodeOutput, requiredServices []string) (bool, error) {
	// Build health check script
	healthCheckScript := h.buildHealthCheckScript(requiredServices)

	// Execute health check via SSH
	result, err := h.executeRemoteCommand(node, healthCheckScript)
	if err != nil {
		return false, err
	}

	// Parse results and update service statuses
	h.mu.Lock()
	status := h.statuses[node.Name]

	// Parse service status from output
	for _, service := range requiredServices {
		status.Services[service] = h.isServiceHealthy(result, service)
	}
	h.mu.Unlock()

	// Check if all required services are healthy
	allHealthy := true
	for _, service := range requiredServices {
		if !status.Services[service] {
			allHealthy = false
			break
		}
	}

	return allHealthy, nil
}

// buildHealthCheckScript creates the health check script based on required services
func (h *HealthChecker) buildHealthCheckScript(services []string) string {
	script := `#!/bin/bash
set -e

echo "=== Node Health Check ==="
echo "Timestamp: $(date)"
echo ""

# Function to check service
check_service() {
    local service=$1
    if systemctl is-active --quiet $service; then
        echo "SERVICE:$service:RUNNING"
    else
        echo "SERVICE:$service:STOPPED"
    fi
}

# Function to check command
check_command() {
    local cmd=$1
    if command -v $cmd &> /dev/null; then
        echo "COMMAND:$cmd:AVAILABLE"
    else
        echo "COMMAND:$cmd:MISSING"
    fi
}

# Function to check port
check_port() {
    local port=$1
    if netstat -tuln | grep -q ":$port "; then
        echo "PORT:$port:LISTENING"
    else
        echo "PORT:$port:CLOSED"
    fi
}

# Basic system checks
echo "UPTIME:$(uptime -p)"
echo "LOAD:$(cat /proc/loadavg | cut -d' ' -f1-3)"
echo "MEMORY:$(free -m | grep Mem | awk '{print $3"/"$2" MB"}')"
echo "DISK:$(df -h / | tail -1 | awk '{print $3"/"$2" ("$5")"}')"
echo ""

# Check required services
`

	// Add checks for each required service
	for _, service := range services {
		switch service {
		case "docker":
			script += `
# Docker checks
check_service docker
check_command docker
docker version &>/dev/null && echo "DOCKER:VERSION:OK" || echo "DOCKER:VERSION:FAIL"
docker ps &>/dev/null && echo "DOCKER:PS:OK" || echo "DOCKER:PS:FAIL"
`
		case "wireguard":
			script += `
# WireGuard checks
check_command wg
if [ -f /etc/wireguard/wg0.conf ]; then
    echo "WIREGUARD:CONFIG:EXISTS"
    wg show wg0 &>/dev/null && echo "WIREGUARD:INTERFACE:UP" || echo "WIREGUARD:INTERFACE:DOWN"
else
    echo "WIREGUARD:CONFIG:MISSING"
fi
`
		case "kubernetes":
			script += `
# Kubernetes checks
check_command kubectl
kubectl version --client &>/dev/null && echo "KUBECTL:VERSION:OK" || echo "KUBECTL:VERSION:FAIL"
if [ -f /root/kube_config_cluster.yml ]; then
    echo "KUBECONFIG:EXISTS"
    export KUBECONFIG=/root/kube_config_cluster.yml
    kubectl get nodes &>/dev/null && echo "KUBERNETES:API:OK" || echo "KUBERNETES:API:FAIL"
else
    echo "KUBECONFIG:MISSING"
fi
`
		case "kubelet":
			script += `
# Kubelet checks
check_service kubelet
check_port 10250
`
		case "etcd":
			script += `
# Etcd checks
check_port 2379
check_port 2380
`
		case "nginx":
			script += `
# NGINX checks
kubectl get svc -n ingress-nginx nginx-ingress-controller &>/dev/null && echo "NGINX:SERVICE:OK" || echo "NGINX:SERVICE:FAIL"
kubectl get pods -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx &>/dev/null && echo "NGINX:PODS:OK" || echo "NGINX:PODS:FAIL"
`
		case "ssh":
			script += `
# SSH checks
check_service ssh
check_port 22
`
		}
	}

	script += `
echo ""
echo "=== Health Check Complete ==="
`

	return script
}

// executeRemoteCommand executes a command on a remote node
func (h *HealthChecker) executeRemoteCommand(node *providers.NodeOutput, script string) (string, error) {
	// This is a simplified version - in production, you'd use the actual SSH execution
	// For now, we'll simulate it with a placeholder

	cmd, err := remote.NewCommand(h.ctx, fmt.Sprintf("health-check-%s-%d", node.Name, time.Now().Unix()), &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       node.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(node.SSHUser),
			PrivateKey: pulumi.String(h.getSSHPrivateKey()),
		},
		Create: pulumi.String(script),
	}, pulumi.IgnoreChanges([]string{"create"}))

	if err != nil {
		return "", err
	}

	// Get the output
	output := cmd.Stdout.ApplyT(func(out string) string {
		return out
	}).(pulumi.StringOutput)

	// Wait for output to be available
	result := ""
	output.ApplyT(func(out string) string {
		result = out
		return out
	})

	return result, nil
}

// isServiceHealthy checks if a service is healthy based on output
func (h *HealthChecker) isServiceHealthy(output string, service string) bool {
	// Parse output to determine if service is healthy
	// This is a simplified check - in production you'd parse more thoroughly

	switch service {
	case "docker":
		return contains(output, "DOCKER:PS:OK") && contains(output, "SERVICE:docker:RUNNING")
	case "wireguard":
		return contains(output, "WIREGUARD:INTERFACE:UP")
	case "kubernetes":
		return contains(output, "KUBERNETES:API:OK")
	case "kubelet":
		return contains(output, "SERVICE:kubelet:RUNNING")
	case "nginx":
		return contains(output, "NGINX:SERVICE:OK") && contains(output, "NGINX:PODS:OK")
	default:
		return contains(output, fmt.Sprintf("SERVICE:%s:RUNNING", service))
	}
}

// allNodesReady checks if all nodes are ready
func (h *HealthChecker) allNodesReady() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, status := range h.statuses {
		if !status.IsHealthy {
			return false
		}
	}
	return len(h.statuses) > 0
}

// reportStatus periodically reports the status of all nodes
func (h *HealthChecker) reportStatus(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()

			readyCount := 0
			//			totalCount := len(h.statuses)

			statusMessages := []string{}
			for name, status := range h.statuses {
				if status.IsHealthy {
					readyCount++
					statusMessages = append(statusMessages, fmt.Sprintf("✓ %s", name))
				} else {
					statusMessages = append(statusMessages, fmt.Sprintf("✗ %s", name))
				}
			}

			h.mu.RUnlock()

			h.ctx.Log.Info("Health check status", nil)
		}
	}
}

// getSSHPrivateKey gets the SSH private key
func (h *HealthChecker) getSSHPrivateKey() string {
	// In production, this would read the actual key
	return "SSH_PRIVATE_KEY_CONTENT"
}

// GetNodeStatus returns the current status of a specific node
func (h *HealthChecker) GetNodeStatus(nodeName string) (*NodeStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status, exists := h.statuses[nodeName]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeName)
	}

	return status, nil
}

// GetAllStatuses returns the current status of all nodes
func (h *HealthChecker) GetAllStatuses() map[string]*NodeStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]*NodeStatus)
	for k, v := range h.statuses {
		result[k] = v
	}

	return result
}

// WaitForKubernetesReady waits specifically for Kubernetes to be ready
func (h *HealthChecker) WaitForKubernetesReady() error {
	requiredServices := []string{
		"docker",
		"kubelet",
		"kubernetes",
		"etcd",
	}

	h.ctx.Log.Info("Waiting for Kubernetes cluster to be ready", nil)
	return h.WaitForNodesReady(requiredServices)
}

// WaitForIngressReady waits for ingress controller to be ready
func (h *HealthChecker) WaitForIngressReady() error {
	requiredServices := []string{
		"nginx",
		"kubernetes",
	}

	h.ctx.Log.Info("Waiting for NGINX Ingress to be ready", nil)
	return h.WaitForNodesReady(requiredServices)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isRecoverableError(err error) bool {
	// Define which errors are recoverable and should trigger retry
	// For now, most errors are considered recoverable
	return err != nil
}
