package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// DNSRecordComponent represents a single DNS record
type DNSRecordComponent struct {
	pulumi.ResourceState

	RecordName pulumi.StringOutput `pulumi:"recordName"`
	RecordType pulumi.StringOutput `pulumi:"recordType"`
	Value      pulumi.StringOutput `pulumi:"value"`
	TTL        pulumi.IntOutput    `pulumi:"ttl"`
	Status     pulumi.StringOutput `pulumi:"status"`
}

// NewDNSComponentGranular creates DNS components with individual record components
func NewDNSComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, opts ...pulumi.ResourceOption) (*DNSComponent, error) {
	component := &DNSComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:dns:DNS", name, component, opts...)
	if err != nil {
		return nil, err
	}

	domain := config.Network.DNS.Domain
	if domain == "" {
		domain = "chalkan3.com.br"
	}

	dnsRecords := make(map[string]interface{})

	// Create individual DNS record components for each node
	nodes.ApplyT(func(nodeList []interface{}) error {
		// API endpoint DNS record
		apiRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-api", name),
			"api", domain, "A", "pending-api-ip", 300, component)
		if err == nil {
			dnsRecords["api"] = apiRecord
		}

		// Ingress DNS record
		ingressRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-ingress", name),
			"kube-ingress", domain, "A", "pending-ingress-ip", 300, component)
		if err == nil {
			dnsRecords["ingress"] = ingressRecord
		}

		// Wildcard ingress DNS record
		wildcardRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-wildcard-ingress", name),
			"*.kube-ingress", domain, "A", "pending-ingress-ip", 300, component)
		if err == nil {
			dnsRecords["wildcard"] = wildcardRecord
		}

		// Create DNS record for each node
		for i := 0; i < len(nodeList) && i < 6; i++ {
			nodeName := fmt.Sprintf("node-%d", i+1)
			nodeRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-%s", name, nodeName),
				nodeName, domain, "A", fmt.Sprintf("pending-node-%d-ip", i+1), 300, component)
			if err == nil {
				dnsRecords[nodeName] = nodeRecord
			}
		}

		// Master nodes DNS records
		for i := 0; i < 3; i++ {
			masterName := fmt.Sprintf("master-%d", i+1)
			masterRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-%s", name, masterName),
				masterName, domain, "A", fmt.Sprintf("pending-master-%d-ip", i+1), 300, component)
			if err == nil {
				dnsRecords[masterName] = masterRecord
			}
		}

		// Worker nodes DNS records
		for i := 0; i < 3; i++ {
			workerName := fmt.Sprintf("worker-%d", i+1)
			workerRecord, err := newDNSRecordComponent(ctx, fmt.Sprintf("%s-%s", name, workerName),
				workerName, domain, "A", fmt.Sprintf("pending-worker-%d-ip", i+1), 300, component)
			if err == nil {
				dnsRecords[workerName] = workerRecord
			}
		}

		return nil
	})

	// Set component outputs
	component.Records = pulumi.Map{
		"domain":       pulumi.String(domain),
		"provider":     pulumi.String(config.Network.DNS.Provider),
		"totalRecords": pulumi.Int(len(dnsRecords)),
		"status":       pulumi.String("configured"),
	}.ToMapOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"records": component.Records,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newDNSRecordComponent creates a single DNS record component
func newDNSRecordComponent(ctx *pulumi.Context, name, recordName, domain, recordType, value string, ttl int, parent pulumi.Resource) (*DNSRecordComponent, error) {
	component := &DNSRecordComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:dns:Record", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.RecordName = pulumi.Sprintf("%s.%s", recordName, domain).ToStringOutput()
	component.RecordType = pulumi.String(recordType).ToStringOutput()
	component.Value = pulumi.String(value).ToStringOutput()
	component.TTL = pulumi.Int(ttl).ToIntOutput()
	component.Status = pulumi.String("pending-creation").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"recordName": component.RecordName,
		"recordType": component.RecordType,
		"value":      component.Value,
		"ttl":        component.TTL,
		"status":     component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
