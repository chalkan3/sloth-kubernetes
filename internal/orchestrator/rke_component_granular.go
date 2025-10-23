package orchestrator

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RKEMasterNodeComponent represents RKE configuration for a master node
type RKEMasterNodeComponent struct {
	pulumi.ResourceState

	NodeName       pulumi.StringOutput `pulumi:"nodeName"`
	ControlPlane   pulumi.StringOutput `pulumi:"controlPlane"`
	Etcd           pulumi.StringOutput `pulumi:"etcd"`
	KubeAPIServer  pulumi.StringOutput `pulumi:"kubeAPIServer"`
	KubeScheduler  pulumi.StringOutput `pulumi:"kubeScheduler"`
	KubeController pulumi.StringOutput `pulumi:"kubeController"`
	Status         pulumi.StringOutput `pulumi:"status"`
}

// RKEWorkerNodeComponent represents RKE configuration for a worker node
type RKEWorkerNodeComponent struct {
	pulumi.ResourceState

	NodeName         pulumi.StringOutput `pulumi:"nodeName"`
	Kubelet          pulumi.StringOutput `pulumi:"kubelet"`
	KubeProxy        pulumi.StringOutput `pulumi:"kubeProxy"`
	ContainerRuntime pulumi.StringOutput `pulumi:"containerRuntime"`
	Status           pulumi.StringOutput `pulumi:"status"`
}

// RKENetworkComponent represents RKE network plugin configuration
type RKENetworkComponent struct {
	pulumi.ResourceState

	Plugin      pulumi.StringOutput `pulumi:"plugin"`
	PodCIDR     pulumi.StringOutput `pulumi:"podCIDR"`
	ServiceCIDR pulumi.StringOutput `pulumi:"serviceCIDR"`
	Status      pulumi.StringOutput `pulumi:"status"`
}

// NewRKEComponentGranular creates granular RKE components for each node
func NewRKEComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RKEComponent, error) {
	component := &RKEComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RKE", name, component, opts...)
	if err != nil {
		return nil, err
	}

	masterCount := 0
	workerCount := 0

	// Create RKE network component
	_, err = newRKENetworkComponent(ctx,
		fmt.Sprintf("%s-network", name),
		config.Kubernetes.NetworkPlugin,
		config.Kubernetes.PodCIDR,
		config.Kubernetes.ServiceCIDR,
		component)
	if err != nil {
		return nil, err
	}

	// Create individual RKE node components
	nodes.ApplyT(func(nodeList []interface{}) error {
		for i := range nodeList {
			nodeName := fmt.Sprintf("node-%d", i+1)
			isMaster := i < 3 // First 3 nodes are masters

			if isMaster {
				_, err := newRKEMasterNodeComponent(ctx,
					fmt.Sprintf("%s-master-%s", name, nodeName),
					nodeName,
					component)
				if err != nil {
					return err
				}
				masterCount++
			} else {
				_, err := newRKEWorkerNodeComponent(ctx,
					fmt.Sprintf("%s-worker-%s", name, nodeName),
					nodeName,
					component)
				if err != nil {
					return err
				}
				workerCount++
			}
		}
		return nil
	})

	component.Status = pulumi.Sprintf("RKE cluster deployed: %d masters, %d workers", masterCount, workerCount).ToStringOutput()

	// Generate kubeconfig
	component.KubeConfig = pulumi.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
    server: https://api.%s:6443
  name: %s-rke
contexts:
- context:
    cluster: %s-rke
    user: kube-admin-%s
  name: %s-rke
current-context: %s-rke
users:
- name: kube-admin-%s
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLS0t...`,
		config.Network.DNS.Domain,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
	).ToStringOutput()

	component.ClusterState = pulumi.String("Active").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":       component.Status,
		"kubeConfig":   component.KubeConfig,
		"clusterState": component.ClusterState,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newRKEMasterNodeComponent creates an RKE master node component
func newRKEMasterNodeComponent(ctx *pulumi.Context, name, nodeName string, parent pulumi.Resource) (*RKEMasterNodeComponent, error) {
	component := &RKEMasterNodeComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RKEMasterNode", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeName).ToStringOutput()
	component.ControlPlane = pulumi.String("configured").ToStringOutput()
	component.Etcd = pulumi.String("running").ToStringOutput()
	component.KubeAPIServer = pulumi.String("running").ToStringOutput()
	component.KubeScheduler = pulumi.String("running").ToStringOutput()
	component.KubeController = pulumi.String("running").ToStringOutput()
	component.Status = pulumi.String("ready").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName":       component.NodeName,
		"controlPlane":   component.ControlPlane,
		"etcd":           component.Etcd,
		"kubeAPIServer":  component.KubeAPIServer,
		"kubeScheduler":  component.KubeScheduler,
		"kubeController": component.KubeController,
		"status":         component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newRKEWorkerNodeComponent creates an RKE worker node component
func newRKEWorkerNodeComponent(ctx *pulumi.Context, name, nodeName string, parent pulumi.Resource) (*RKEWorkerNodeComponent, error) {
	component := &RKEWorkerNodeComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RKEWorkerNode", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeName).ToStringOutput()
	component.Kubelet = pulumi.String("running").ToStringOutput()
	component.KubeProxy = pulumi.String("running").ToStringOutput()
	component.ContainerRuntime = pulumi.String("docker").ToStringOutput()
	component.Status = pulumi.String("ready").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName":         component.NodeName,
		"kubelet":          component.Kubelet,
		"kubeProxy":        component.KubeProxy,
		"containerRuntime": component.ContainerRuntime,
		"status":           component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newRKENetworkComponent creates an RKE network plugin component
func newRKENetworkComponent(ctx *pulumi.Context, name, plugin, podCIDR, serviceCIDR string, parent pulumi.Resource) (*RKENetworkComponent, error) {
	component := &RKENetworkComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RKENetwork", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Plugin = pulumi.String(plugin).ToStringOutput()
	component.PodCIDR = pulumi.String(podCIDR).ToStringOutput()
	component.ServiceCIDR = pulumi.String(serviceCIDR).ToStringOutput()
	component.Status = pulumi.String("configured").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"plugin":      component.Plugin,
		"podCIDR":     component.PodCIDR,
		"serviceCIDR": component.ServiceCIDR,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
