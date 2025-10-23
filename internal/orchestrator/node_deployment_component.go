package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// NodeDeploymentComponent handles node deployment
type NodeDeploymentComponent struct {
	pulumi.ResourceState

	Nodes  pulumi.ArrayOutput  `pulumi:"nodes"`
	Status pulumi.StringOutput `pulumi:"status"`
}

// IndividualNodeComponent represents a single node
type IndividualNodeComponent struct {
	pulumi.ResourceState

	NodeName    pulumi.StringOutput `pulumi:"nodeName"`
	Provider    pulumi.StringOutput `pulumi:"provider"`
	Region      pulumi.StringOutput `pulumi:"region"`
	Size        pulumi.StringOutput `pulumi:"size"`
	PublicIP    pulumi.StringOutput `pulumi:"publicIP"`
	PrivateIP   pulumi.StringOutput `pulumi:"privateIP"`
	WireGuardIP pulumi.StringOutput `pulumi:"wireGuardIP"`
	Roles       pulumi.ArrayOutput  `pulumi:"roles"`
	Status      pulumi.StringOutput `pulumi:"status"`
}

// NewNodeDeploymentComponent creates a new node deployment component with individual node components
func NewNodeDeploymentComponent(ctx *pulumi.Context, name string, clusterConfig *config.ClusterConfig, providerMap pulumi.MapOutput, opts ...pulumi.ResourceOption) (*NodeDeploymentComponent, error) {
	component := &NodeDeploymentComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:compute:NodeDeployment", name, component, opts...)
	if err != nil {
		return nil, err
	}

	nodeComponents := []*IndividualNodeComponent{}
	nodesArray := []pulumi.Output{}

	// Add individual nodes from Nodes section
	for _, nodeConfig := range clusterConfig.Nodes {
		nodeComp, err := newIndividualNodeComponent(ctx, fmt.Sprintf("%s-%s", name, nodeConfig.Name), &nodeConfig, component)
		if err != nil {
			return nil, err
		}
		nodeComponents = append(nodeComponents, nodeComp)
		nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
	}

	// Add nodes from pools - each node gets its own component
	nodeIndex := len(nodeComponents)
	for poolName, poolConfig := range clusterConfig.NodePools {
		for i := 0; i < poolConfig.Count; i++ {
			nodeName := fmt.Sprintf("%s-%d", poolName, i+1)

			// Create a node config from pool config
			nodeConfig := config.NodeConfig{
				Name:        nodeName,
				Provider:    poolConfig.Provider,
				Region:      poolConfig.Region,
				Size:        poolConfig.Size,
				Image:       poolConfig.Image,
				Roles:       poolConfig.Roles,
				Labels:      poolConfig.Labels,
				Taints:      poolConfig.Taints,
				PrivateIP:   fmt.Sprintf("10.0.1.%d", nodeIndex+1),
				WireGuardIP: fmt.Sprintf("10.8.0.%d", 10+nodeIndex),
			}

			nodeComp, err := newIndividualNodeComponent(ctx, fmt.Sprintf("%s-%s-%s", name, poolName, nodeName), &nodeConfig, component)
			if err != nil {
				return nil, err
			}
			nodeComponents = append(nodeComponents, nodeComp)
			nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
			nodeIndex++
		}
	}

	// Set component outputs
	component.Nodes = pulumi.ToArrayOutput(nodesArray)
	component.Status = pulumi.Sprintf("Deployed %d individual node components", len(nodeComponents))

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodes":  component.Nodes,
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newIndividualNodeComponent creates a component for a single node
func newIndividualNodeComponent(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, parent pulumi.Resource) (*IndividualNodeComponent, error) {
	component := &IndividualNodeComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:compute:Node", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	// Set node properties
	component.NodeName = pulumi.String(nodeConfig.Name).ToStringOutput()
	component.Provider = pulumi.String(nodeConfig.Provider).ToStringOutput()
	component.Region = pulumi.String(nodeConfig.Region).ToStringOutput()
	component.Size = pulumi.String(nodeConfig.Size).ToStringOutput()
	component.PublicIP = pulumi.Sprintf("pending-%s", nodeConfig.Name).ToStringOutput()
	component.PrivateIP = pulumi.String(nodeConfig.PrivateIP).ToStringOutput()
	component.WireGuardIP = pulumi.String(nodeConfig.WireGuardIP).ToStringOutput()

	// Convert roles to pulumi array
	rolesArray := make([]pulumi.Output, len(nodeConfig.Roles))
	for i, role := range nodeConfig.Roles {
		rolesArray[i] = pulumi.String(role).ToStringOutput()
	}
	component.Roles = pulumi.ToArrayOutput(rolesArray)

	component.Status = pulumi.String("configured").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName":    component.NodeName,
		"provider":    component.Provider,
		"region":      component.Region,
		"size":        component.Size,
		"publicIP":    component.PublicIP,
		"privateIP":   component.PrivateIP,
		"wireGuardIP": component.WireGuardIP,
		"roles":       component.Roles,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
