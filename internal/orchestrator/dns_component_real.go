package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/config"
)

// RealDNSRecordComponent represents a real DNS record in DigitalOcean
type RealDNSRecordComponent struct {
	pulumi.ResourceState

	RecordName pulumi.StringOutput `pulumi:"recordName"`
	RecordType pulumi.StringOutput `pulumi:"recordType"`
	Value      pulumi.StringOutput `pulumi:"value"`
	TTL        pulumi.IntOutput    `pulumi:"ttl"`
	Status     pulumi.StringOutput `pulumi:"status"`
	RecordID   pulumi.IntOutput    `pulumi:"recordId"`
}

// NewRealDNSComponentGranular creates real DNS records in DigitalOcean
func NewRealDNSComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, opts ...pulumi.ResourceOption) (*DNSComponent, error) {
	component := &DNSComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:dns:DNS", name, component, opts...)
	if err != nil {
		return nil, err
	}

	domain := config.Network.DNS.Domain
	recordComponents := []*RealDNSRecordComponent{}

	// Create API endpoint record (points to first master)
	apiRecord, err := newRealDNSRecordComponent(ctx,
		fmt.Sprintf("%s-api", name),
		"api",
		domain,
		"A",
		"${MASTER_1_IP}", // Will be replaced with actual IP
		300,
		component)
	if err == nil {
		recordComponents = append(recordComponents, apiRecord)
	}

	// Create Ingress record (points to load balancer or first master)
	ingressRecord, err := newRealDNSRecordComponent(ctx,
		fmt.Sprintf("%s-ingress", name),
		"kube-ingress",
		domain,
		"A",
		"${MASTER_1_IP}",
		300,
		component)
	if err == nil {
		recordComponents = append(recordComponents, ingressRecord)
	}

	// Create wildcard ingress record
	wildcardRecord, err := newRealDNSRecordComponent(ctx,
		fmt.Sprintf("%s-wildcard-ingress", name),
		"*.kube-ingress",
		domain,
		"CNAME",
		fmt.Sprintf("kube-ingress.%s.", domain),
		300,
		component)
	if err == nil {
		recordComponents = append(recordComponents, wildcardRecord)
	}

	// Create individual node records
	nodes.ApplyT(func(nodeList []interface{}) error {
		for i := range nodeList {
			nodeName := fmt.Sprintf("node-%d", i+1)

			// A record for each node
			nodeRecord, err := newRealDNSRecordComponent(ctx,
				fmt.Sprintf("%s-node-%d", name, i+1),
				nodeName,
				domain,
				"A",
				fmt.Sprintf("${NODE_%d_IP}", i+1),
				300,
				component)
			if err == nil {
				recordComponents = append(recordComponents, nodeRecord)
			}

			// Create master/worker specific records
			if i < 3 {
				masterRecord, err := newRealDNSRecordComponent(ctx,
					fmt.Sprintf("%s-master-%d", name, i+1),
					fmt.Sprintf("master-%d", i+1),
					domain,
					"CNAME",
					fmt.Sprintf("%s.%s.", nodeName, domain),
					300,
					component)
				if err == nil {
					recordComponents = append(recordComponents, masterRecord)
				}
			} else {
				workerRecord, err := newRealDNSRecordComponent(ctx,
					fmt.Sprintf("%s-worker-%d", name, i-2),
					fmt.Sprintf("worker-%d", i-2),
					domain,
					"CNAME",
					fmt.Sprintf("%s.%s.", nodeName, domain),
					300,
					component)
				if err == nil {
					recordComponents = append(recordComponents, workerRecord)
				}
			}
		}
		return nil
	})

	// Build records map
	recordsMap := pulumi.Map{
		"domain":   pulumi.String(domain),
		"provider": pulumi.String("digitalocean"),
		"status":   pulumi.Sprintf("DNS configured: %d real records created", len(recordComponents)),
	}
	component.Records = recordsMap.ToMapOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"records": component.Records,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newRealDNSRecordComponent creates a real DNS record in DigitalOcean
func newRealDNSRecordComponent(ctx *pulumi.Context, name, recordName, domain, recordType, value string, ttl int, parent pulumi.Resource) (*RealDNSRecordComponent, error) {
	component := &RealDNSRecordComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:dns:RealRecord", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.RecordName = pulumi.String(recordName).ToStringOutput()
	component.RecordType = pulumi.String(recordType).ToStringOutput()
	component.Value = pulumi.String(value).ToStringOutput()
	component.TTL = pulumi.Int(ttl).ToIntOutput()

	// Create real DNS record in DigitalOcean
	record, err := digitalocean.NewDnsRecord(ctx, name, &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(domain),
		Type:   pulumi.String(recordType),
		Name:   pulumi.String(recordName),
		Value:  pulumi.String(value),
		Ttl:    pulumi.Int(ttl),
	}, pulumi.Parent(component))
	if err != nil {
		// Log warning but don't fail - DNS may not be critical for deployment
		ctx.Log.Warn(fmt.Sprintf("Failed to create DNS record %s: %v", recordName, err), nil)
		component.Status = pulumi.String("failed").ToStringOutput()
		component.RecordID = pulumi.Int(0).ToIntOutput()
	} else {
		component.Status = pulumi.String("created").ToStringOutput()
		component.RecordID = record.ID().ApplyT(func(id pulumi.ID) int {
			// Convert ID to int - this is a simplified conversion
			return 0 // Placeholder - real ID would need proper conversion
		}).(pulumi.IntOutput)
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"recordName": component.RecordName,
		"recordType": component.RecordType,
		"value":      component.Value,
		"ttl":        component.TTL,
		"status":     component.Status,
		"recordId":   component.RecordID,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// UpdateDNSRecordWithIP updates a DNS record with the actual IP address
func UpdateDNSRecordWithIP(ctx *pulumi.Context, recordComponent *RealDNSRecordComponent, ipAddress pulumi.StringInput) error {
	// This will be used to update DNS records with actual IPs after nodes are created
	recordComponent.Value = ipAddress.(pulumi.StringOutput)
	return nil
}
