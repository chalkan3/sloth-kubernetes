package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/config"
)

// IngressControllerComponent represents the NGINX ingress controller
type IngressControllerComponent struct {
	pulumi.ResourceState

	Namespace pulumi.StringOutput `pulumi:"namespace"`
	Replicas  pulumi.IntOutput    `pulumi:"replicas"`
	Status    pulumi.StringOutput `pulumi:"status"`
}

// IngressClassComponent represents an ingress class
type IngressClassComponent struct {
	pulumi.ResourceState

	ClassName pulumi.StringOutput `pulumi:"className"`
	IsDefault pulumi.BoolOutput   `pulumi:"isDefault"`
	Status    pulumi.StringOutput `pulumi:"status"`
}

// CertManagerComponent represents cert-manager addon
type CertManagerComponent struct {
	pulumi.ResourceState

	Namespace pulumi.StringOutput `pulumi:"namespace"`
	Version   pulumi.StringOutput `pulumi:"version"`
	Issuer    pulumi.StringOutput `pulumi:"issuer"`
	Status    pulumi.StringOutput `pulumi:"status"`
}

// MetricsServerComponent represents metrics-server addon
type MetricsServerComponent struct {
	pulumi.ResourceState

	Namespace pulumi.StringOutput `pulumi:"namespace"`
	Status    pulumi.StringOutput `pulumi:"status"`
}

// NewIngressComponentGranular creates granular ingress components
func NewIngressComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*IngressComponent, error) {
	component := &IngressComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:ingress:NGINX", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Create NGINX Ingress Controller component
	controller, err := newIngressControllerComponent(ctx,
		fmt.Sprintf("%s-controller", name),
		"ingress-nginx",
		2, // 2 replicas
		component)
	if err != nil {
		return nil, err
	}

	// Create Ingress Class component
	_, err = newIngressClassComponent(ctx,
		fmt.Sprintf("%s-class-nginx", name),
		"nginx",
		true, // default class
		controller)
	if err != nil {
		return nil, err
	}

	// Create cert-manager component
	_, err = newCertManagerComponent(ctx,
		fmt.Sprintf("%s-cert-manager", name),
		"cert-manager",
		"v1.12.0",
		"letsencrypt",
		component)
	if err != nil {
		return nil, err
	}

	component.Status = pulumi.String("NGINX Ingress installed with cert-manager").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newIngressControllerComponent creates an ingress controller component
func newIngressControllerComponent(ctx *pulumi.Context, name, namespace string, replicas int, parent pulumi.Resource) (*IngressControllerComponent, error) {
	component := &IngressControllerComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:ingress:Controller", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Namespace = pulumi.String(namespace).ToStringOutput()
	component.Replicas = pulumi.Int(replicas).ToIntOutput()
	component.Status = pulumi.String("running").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"namespace": component.Namespace,
		"replicas":  component.Replicas,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newIngressClassComponent creates an ingress class component
func newIngressClassComponent(ctx *pulumi.Context, name, className string, isDefault bool, parent pulumi.Resource) (*IngressClassComponent, error) {
	component := &IngressClassComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:ingress:IngressClass", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.ClassName = pulumi.String(className).ToStringOutput()
	component.IsDefault = pulumi.Bool(isDefault).ToBoolOutput()
	component.Status = pulumi.String("created").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"className": component.ClassName,
		"isDefault": component.IsDefault,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newCertManagerComponent creates a cert-manager component
func newCertManagerComponent(ctx *pulumi.Context, name, namespace, version, issuer string, parent pulumi.Resource) (*CertManagerComponent, error) {
	component := &CertManagerComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:addons:CertManager", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Namespace = pulumi.String(namespace).ToStringOutput()
	component.Version = pulumi.String(version).ToStringOutput()
	component.Issuer = pulumi.String(issuer).ToStringOutput()
	component.Status = pulumi.String("installed").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"namespace": component.Namespace,
		"version":   component.Version,
		"issuer":    component.Issuer,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// NewAddonsComponentGranular creates granular addon components
func NewAddonsComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*AddonsComponent, error) {
	component := &AddonsComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:addons:Addons", name, component, opts...)
	if err != nil {
		return nil, err
	}

	addonsInstalled := 0

	// Create metrics-server component
	_, err = newMetricsServerComponent(ctx,
		fmt.Sprintf("%s-metrics-server", name),
		"kube-system",
		component)
	if err != nil {
		return nil, err
	}
	addonsInstalled++

	// Create addon components for each configured addon
	for _, addon := range config.Kubernetes.Addons {
		_, err = newAddonComponent(ctx,
			fmt.Sprintf("%s-addon-%s", name, addon.Name),
			addon.Name,
			addon.Namespace,
			addon.Version,
			component)
		if err != nil {
			return nil, err
		}
		addonsInstalled++
	}

	component.Status = pulumi.Sprintf("Installed %d addons", addonsInstalled).ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newMetricsServerComponent creates a metrics-server component
func newMetricsServerComponent(ctx *pulumi.Context, name, namespace string, parent pulumi.Resource) (*MetricsServerComponent, error) {
	component := &MetricsServerComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:addons:MetricsServer", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Namespace = pulumi.String(namespace).ToStringOutput()
	component.Status = pulumi.String("running").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"namespace": component.Namespace,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// AddonComponent represents a generic addon
type AddonComponent struct {
	pulumi.ResourceState

	Name      pulumi.StringOutput `pulumi:"name"`
	Namespace pulumi.StringOutput `pulumi:"namespace"`
	Version   pulumi.StringOutput `pulumi:"version"`
	Status    pulumi.StringOutput `pulumi:"status"`
}

// newAddonComponent creates a generic addon component
func newAddonComponent(ctx *pulumi.Context, resourceName, name, namespace, version string, parent pulumi.Resource) (*AddonComponent, error) {
	component := &AddonComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:addons:Addon", resourceName, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Name = pulumi.String(name).ToStringOutput()
	component.Namespace = pulumi.String(namespace).ToStringOutput()
	component.Version = pulumi.String(version).ToStringOutput()
	component.Status = pulumi.String("installed").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"name":      component.Name,
		"namespace": component.Namespace,
		"version":   component.Version,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
