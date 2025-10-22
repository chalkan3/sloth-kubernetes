package components

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DNSRealComponent creates REAL DNS records in DigitalOcean
type DNSRealComponent struct {
	pulumi.ResourceState

	Status      pulumi.StringOutput `pulumi:"status"`
	RecordCount pulumi.IntOutput    `pulumi:"recordCount"`
	Domain      pulumi.StringOutput `pulumi:"domain"`
	APIEndpoint pulumi.StringOutput `pulumi:"apiEndpoint"`
}

// NewDNSRealComponent creates real DNS records in DigitalOcean
func NewDNSRealComponent(ctx *pulumi.Context, name string, domain string, nodes []*RealNodeComponent, opts ...pulumi.ResourceOption) (*DNSRealComponent, error) {
	component := &DNSRealComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:dns:DNSReal", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Validate domain is not empty or placeholder
	if domain == "" || domain == "example.com" {
		ctx.Log.Warn("⚠️  DNS domain is empty or placeholder (example.com), skipping DNS record creation", nil)
		component.Status = pulumi.String("skipped: no domain configured").ToStringOutput()
		component.RecordCount = pulumi.Int(0).ToIntOutput()
		component.Domain = pulumi.String("").ToStringOutput()
		component.APIEndpoint = pulumi.String("").ToStringOutput()

		if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
			"status":      component.Status,
			"recordCount": component.RecordCount,
			"domain":      component.Domain,
			"apiEndpoint": component.APIEndpoint,
		}); err != nil {
			return nil, err
		}

		return component, nil
	}

	ctx.Log.Info(fmt.Sprintf("🌐 Creating DNS records for domain: %s", domain), nil)

	recordCount := 0

	// Create A record for API endpoint (points to first master)
	if len(nodes) > 0 {
		firstMaster := nodes[0]

		_, err := digitalocean.NewDnsRecord(ctx, fmt.Sprintf("%s-api", name), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(domain),
			Type:   pulumi.String("A"),
			Name:   pulumi.String("api"),
			Value:  firstMaster.PublicIP,
			Ttl:    pulumi.Int(300),
		}, pulumi.Parent(component))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to create API DNS record: %v", err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ Created DNS: api.%s -> first master", domain), nil)
			recordCount++
		}
	}

	// Create A record for each node
	for i, node := range nodes {
		nodeName := fmt.Sprintf("node-%d", i+1)

		_, err := digitalocean.NewDnsRecord(ctx, fmt.Sprintf("%s-node-%d", name, i+1), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(domain),
			Type:   pulumi.String("A"),
			Name:   pulumi.String(nodeName),
			Value:  node.PublicIP,
			Ttl:    pulumi.Int(300),
		}, pulumi.Parent(component))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to create DNS for node %d: %v", i+1, err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ Created DNS: %s.%s -> node %d", nodeName, domain, i+1), nil)
			recordCount++
		}
	}

	// Create wildcard A record for ingress (points to all worker nodes via round-robin)
	// For simplicity, point to first worker node (node index 3+)
	if len(nodes) >= 4 {
		firstWorker := nodes[3]

		_, err := digitalocean.NewDnsRecord(ctx, fmt.Sprintf("%s-wildcard-ingress", name), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(domain),
			Type:   pulumi.String("A"),
			Name:   pulumi.String("*.kube"),
			Value:  firstWorker.PublicIP,
			Ttl:    pulumi.Int(300),
		}, pulumi.Parent(component))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to create wildcard ingress DNS: %v", err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ Created DNS: *.kube.%s -> first worker", domain), nil)
			recordCount++
		}
	}

	// Create specific ingress record
	if len(nodes) >= 4 {
		firstWorker := nodes[3]

		_, err := digitalocean.NewDnsRecord(ctx, fmt.Sprintf("%s-ingress", name), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(domain),
			Type:   pulumi.String("A"),
			Name:   pulumi.String("kube-ingress"),
			Value:  firstWorker.PublicIP,
			Ttl:    pulumi.Int(300),
		}, pulumi.Parent(component))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to create ingress DNS: %v", err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ Created DNS: kube-ingress.%s -> first worker", domain), nil)
			recordCount++
		}
	}

	component.Status = pulumi.Sprintf("DNS configured: %d records", recordCount)
	component.RecordCount = pulumi.Int(recordCount).ToIntOutput()
	component.Domain = pulumi.String(domain).ToStringOutput()
	component.APIEndpoint = pulumi.Sprintf("https://api.%s:6443", domain)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":      component.Status,
		"recordCount": component.RecordCount,
		"domain":      component.Domain,
		"apiEndpoint": component.APIEndpoint,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("✅ DNS setup COMPLETE: %d records in %s", recordCount, domain), nil)

	return component, nil
}
