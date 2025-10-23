package dns

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

// Manager handles DNS record creation
type Manager struct {
	ctx     *pulumi.Context
	domain  string
	records []*digitalocean.DnsRecord
	nodes   []*providers.NodeOutput
}

// NewManager creates a new DNS manager
func NewManager(ctx *pulumi.Context, domain string) *Manager {
	return &Manager{
		ctx:     ctx,
		domain:  domain,
		records: make([]*digitalocean.DnsRecord, 0),
	}
}

// CreateNodeRecords creates DNS records for all nodes
func (m *Manager) CreateNodeRecords(nodes map[string][]*providers.NodeOutput) error {
	m.ctx.Log.Info("Creating DNS records for nodes", nil)

	// Counter for each node type
	masterCount := 0
	workerCount := 0

	for provider, nodeList := range nodes {
		for _, node := range nodeList {
			m.nodes = append(m.nodes, node)

			// Determine node type from labels
			nodeType := "node"
			if node.Labels != nil {
				if role, ok := node.Labels["role"]; ok {
					nodeType = role
				}
			}

			// Create appropriate DNS names
			var dnsNames []string

			switch nodeType {
			case "master", "controlplane":
				masterCount++
				dnsNames = []string{
					fmt.Sprintf("master%d", masterCount),
					fmt.Sprintf("master%d-%s", masterCount, provider),
					fmt.Sprintf("k8s-master%d", masterCount),
				}

				// Add API endpoint for first master
				if masterCount == 1 {
					dnsNames = append(dnsNames, "api", "k8s-api")
				}

			case "worker":
				workerCount++
				dnsNames = []string{
					fmt.Sprintf("worker%d", workerCount),
					fmt.Sprintf("worker%d-%s", workerCount, provider),
					fmt.Sprintf("k8s-worker%d", workerCount),
				}

			default:
				dnsNames = []string{
					node.Name,
					fmt.Sprintf("%s-%s", node.Name, provider),
				}
			}

			// Also add the actual node name
			dnsNames = append(dnsNames, node.Name)

			// Create A records for public IPs
			for _, name := range dnsNames {
				if err := m.createARecord(name, node.PublicIP); err != nil {
					return fmt.Errorf("failed to create DNS record for %s: %w", name, err)
				}
			}

			// Create private DNS records (using subdomain)
			if err := m.createARecord(fmt.Sprintf("private-%s", node.Name), node.PrivateIP); err != nil {
				return fmt.Errorf("failed to create private DNS record for %s: %w", node.Name, err)
			}

			// Create WireGuard DNS records
			if node.WireGuardIP != "" {
				wgName := fmt.Sprintf("wg-%s", node.Name)
				if err := m.createARecord(wgName, pulumi.String(node.WireGuardIP)); err != nil {
					return fmt.Errorf("failed to create WireGuard DNS record for %s: %w", node.Name, err)
				}

				// Also create numbered WireGuard records
				if nodeType == "master" || nodeType == "controlplane" {
					if err := m.createARecord(fmt.Sprintf("wg-master%d", masterCount), pulumi.String(node.WireGuardIP)); err != nil {
						return err
					}
				} else if nodeType == "worker" {
					if err := m.createARecord(fmt.Sprintf("wg-worker%d", workerCount), pulumi.String(node.WireGuardIP)); err != nil {
						return err
					}
				}
			}
		}
	}

	// Create wildcard record for ingress (will be updated later with actual ingress IP)
	if err := m.createWildcardRecord(); err != nil {
		return fmt.Errorf("failed to create wildcard record: %w", err)
	}

	m.ctx.Log.Info("DNS records created successfully", nil)

	return nil
}

// createARecord creates an A record
func (m *Manager) createARecord(name string, ip pulumi.StringInput) error {
	recordName := strings.ToLower(name)

	record, err := digitalocean.NewDnsRecord(m.ctx, fmt.Sprintf("dns-%s", recordName), &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(m.domain),
		Type:   pulumi.String("A"),
		Name:   pulumi.String(recordName),
		Value:  ip,
		Ttl:    pulumi.Int(300), // 5 minutes TTL for easier updates
	})
	if err != nil {
		return err
	}

	m.records = append(m.records, record)

	// Export the DNS record
	m.ctx.Export(fmt.Sprintf("dns_%s", strings.ReplaceAll(recordName, "-", "_")),
		pulumi.Sprintf("%s.%s", recordName, m.domain))

	return nil
}

// createWildcardRecord creates a wildcard DNS record for ingress
func (m *Manager) createWildcardRecord() error {
	// Initially point to first worker or master node
	// This will be updated when ingress is installed
	var initialIP pulumi.StringOutput

	for _, node := range m.nodes {
		if node.Labels != nil {
			if role, ok := node.Labels["role"]; ok && role == "worker" {
				initialIP = node.PublicIP
				break
			}
		}
	}

	// If no worker found, use first master
	if initialIP == pulumi.String("").ToStringOutput() && len(m.nodes) > 0 {
		initialIP = m.nodes[0].PublicIP
	}

	if initialIP == pulumi.String("").ToStringOutput() {
		return fmt.Errorf("no nodes available for wildcard record")
	}

	// Create wildcard record for all ingress subdomains
	wildcardRecord, err := digitalocean.NewDnsRecord(m.ctx, "dns-wildcard-ingress", &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(m.domain),
		Type:   pulumi.String("A"),
		Name:   pulumi.String("*.k8s"),
		Value:  initialIP,
		Ttl:    pulumi.Int(300),
	})
	if err != nil {
		return err
	}
	m.records = append(m.records, wildcardRecord)

	// Create specific ingress record
	ingressRecord, err := digitalocean.NewDnsRecord(m.ctx, "dns-kube-ingress", &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(m.domain),
		Type:   pulumi.String("A"),
		Name:   pulumi.String("kube-ingress"),
		Value:  initialIP,
		Ttl:    pulumi.Int(300),
	})
	if err != nil {
		return err
	}
	m.records = append(m.records, ingressRecord)

	m.ctx.Export("ingress_domain", pulumi.String(fmt.Sprintf("kube-ingress.%s", m.domain)))
	m.ctx.Export("wildcard_domain", pulumi.String(fmt.Sprintf("*.k8s.%s", m.domain)))

	return nil
}

// UpdateIngressRecord updates the DNS record for ingress after load balancer is created
func (m *Manager) UpdateIngressRecord(ingressIP pulumi.StringOutput) error {
	// Create or update the main ingress record
	_, err := digitalocean.NewDnsRecord(m.ctx, "dns-ingress-lb", &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(m.domain),
		Type:   pulumi.String("A"),
		Name:   pulumi.String("kube-ingress"),
		Value:  ingressIP,
		Ttl:    pulumi.Int(300),
	}, pulumi.ReplaceOnChanges([]string{"value"}))
	if err != nil {
		return fmt.Errorf("failed to update ingress DNS record: %w", err)
	}

	// Update wildcard record
	_, err = digitalocean.NewDnsRecord(m.ctx, "dns-wildcard-lb", &digitalocean.DnsRecordArgs{
		Domain: pulumi.String(m.domain),
		Type:   pulumi.String("A"),
		Name:   pulumi.String("*.k8s"),
		Value:  ingressIP,
		Ttl:    pulumi.Int(300),
	}, pulumi.ReplaceOnChanges([]string{"value"}))
	if err != nil {
		return fmt.Errorf("failed to update wildcard DNS record: %w", err)
	}

	// Create additional ingress subdomains
	ingressSubdomains := []string{
		"grafana",
		"prometheus",
		"alertmanager",
		"dashboard",
		"argocd",
		"jenkins",
		"gitlab",
		"registry",
	}

	for _, subdomain := range ingressSubdomains {
		_, err = digitalocean.NewDnsRecord(m.ctx, fmt.Sprintf("dns-%s", subdomain), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(m.domain),
			Type:   pulumi.String("A"),
			Name:   pulumi.String(fmt.Sprintf("%s.k8s", subdomain)),
			Value:  ingressIP,
			Ttl:    pulumi.Int(300),
		})
		if err != nil {
			// Log warning but don't fail
			m.ctx.Log.Warn("Failed to create DNS record", nil)
		}
	}

	return nil
}

// CreateClusterRecords creates convenience DNS records for the cluster
func (m *Manager) CreateClusterRecords() error {
	// Create CNAME records for convenience
	conveniences := map[string]string{
		"k8s":        "api",
		"kubernetes": "api",
		"cluster":    "api",
		"rancher":    "kube-ingress",
		"dashboard":  "kube-ingress",
	}

	for name, target := range conveniences {
		_, err := digitalocean.NewDnsRecord(m.ctx, fmt.Sprintf("dns-cname-%s", name), &digitalocean.DnsRecordArgs{
			Domain: pulumi.String(m.domain),
			Type:   pulumi.String("CNAME"),
			Name:   pulumi.String(name),
			Value:  pulumi.String(fmt.Sprintf("%s.%s.", target, m.domain)),
			Ttl:    pulumi.Int(300),
		})
		if err != nil {
			m.ctx.Log.Warn("Failed to create CNAME record", nil)
		}
	}

	return nil
}

// ExportDNSInfo exports DNS information
func (m *Manager) ExportDNSInfo() {
	dnsInfo := make(map[string]interface{})

	// Basic info
	dnsInfo["domain"] = m.domain
	dnsInfo["ingress_url"] = fmt.Sprintf("https://kube-ingress.%s", m.domain)
	dnsInfo["api_url"] = fmt.Sprintf("https://api.%s:6443", m.domain)
	dnsInfo["wildcard"] = fmt.Sprintf("*.k8s.%s", m.domain)

	// Node DNS names
	nodeNames := []string{}
	for i := 1; i <= 3; i++ {
		nodeNames = append(nodeNames, fmt.Sprintf("master%d.%s", i, m.domain))
	}
	for i := 1; i <= 3; i++ {
		nodeNames = append(nodeNames, fmt.Sprintf("worker%d.%s", i, m.domain))
	}
	dnsInfo["nodes"] = nodeNames

	// Service URLs
	dnsInfo["services"] = map[string]string{
		"grafana":    fmt.Sprintf("https://grafana.k8s.%s", m.domain),
		"prometheus": fmt.Sprintf("https://prometheus.k8s.%s", m.domain),
		"dashboard":  fmt.Sprintf("https://dashboard.k8s.%s", m.domain),
	}

	m.ctx.Export("dns_info", pulumi.ToMap(dnsInfo))
}

// GetDomain returns the configured domain
func (m *Manager) GetDomain() string {
	return m.domain
}
