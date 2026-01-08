// Package config provides Lisp-based configuration loading
package config

import (
	"fmt"
)

// LoadFromLisp loads cluster configuration from a Lisp file
func LoadFromLisp(filePath string) (*ClusterConfig, error) {
	expr, err := ParseLispFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Lisp file: %w", err)
	}

	list, ok := expr.(*List)
	if !ok {
		return nil, fmt.Errorf("expected a list at root level")
	}

	head := list.Head()
	if head == nil || head.AsString() != "cluster" {
		return nil, fmt.Errorf("expected (cluster ...) at root level, got: %v", head)
	}

	cfg := &ClusterConfig{
		NodePools: make(map[string]NodePool),
	}

	// Parse each section
	for _, item := range list.Tail() {
		if section, ok := item.(*List); ok {
			if sectionHead := section.Head(); sectionHead != nil {
				switch sectionHead.AsString() {
				case "metadata":
					cfg.Metadata = parseMetadata(section)
				case "cluster":
					cfg.Cluster = parseClusterSpec(section)
				case "providers":
					cfg.Providers = parseProviders(section)
				case "network":
					cfg.Network = parseNetwork(section)
				case "security":
					cfg.Security = parseSecurity(section)
				case "nodes":
					cfg.Nodes = parseNodes(section)
				case "node-pools", "nodePools":
					cfg.NodePools = parseNodePools(section)
				case "kubernetes":
					cfg.Kubernetes = parseKubernetes(section)
				case "monitoring":
					cfg.Monitoring = parseMonitoring(section)
				case "storage":
					cfg.Storage = parseStorage(section)
				case "load-balancer", "loadBalancer":
					cfg.LoadBalancer = parseLoadBalancer(section)
				case "addons":
					cfg.Addons = parseAddons(section)
				case "upgrade":
					cfg.Upgrade = parseUpgradeConfig(section)
				case "backup":
					cfg.Backup = parseBackupConfig(section)
				case "hooks":
					cfg.Hooks = parseHooksConfig(section)
				case "cost-control", "costControl":
					cfg.CostControl = parseCostControlConfig(section)
				case "private-cluster", "privateCluster":
					cfg.PrivateCluster = parsePrivateClusterConfig(section)
				}
			}
		}
	}

	// Apply defaults
	applyDefaults(cfg)

	return cfg, nil
}

func parseMetadata(l *List) Metadata {
	return Metadata{
		Name:        l.GetString("name"),
		Environment: l.GetString("environment"),
		Version:     l.GetString("version"),
		Description: l.GetString("description"),
		Owner:       l.GetString("owner"),
		Team:        l.GetString("team"),
		Labels:      l.GetMap("labels"),
		Annotations: l.GetMap("annotations"),
	}
}

func parseClusterSpec(l *List) ClusterSpec {
	return ClusterSpec{
		Type:             l.GetString("type"),
		Version:          l.GetString("version"),
		Distribution:     l.GetString("distribution"),
		HighAvailability: l.GetBool("high-availability"),
		MultiCloud:       l.GetBool("multi-cloud"),
	}
}

func parseProviders(l *List) ProvidersConfig {
	cfg := ProvidersConfig{}

	for _, item := range l.Tail() {
		if provider, ok := item.(*List); ok {
			if head := provider.Head(); head != nil {
				switch head.AsString() {
				case "digitalocean", "do":
					cfg.DigitalOcean = parseDigitalOceanProvider(provider)
				case "linode":
					cfg.Linode = parseLinodeProvider(provider)
				case "aws":
					cfg.AWS = parseAWSProvider(provider)
				case "azure":
					cfg.Azure = parseAzureProvider(provider)
				case "gcp":
					cfg.GCP = parseGCPProvider(provider)
				case "hetzner":
					cfg.Hetzner = parseHetznerProvider(provider)
				}
			}
		}
	}

	return cfg
}

func parseDigitalOceanProvider(l *List) *DigitalOceanProvider {
	return &DigitalOceanProvider{
		Enabled:    l.GetBool("enabled"),
		Token:      l.GetString("token"),
		Region:     l.GetString("region"),
		SSHKeys:    l.GetStringSlice("ssh-keys"),
		Tags:       l.GetStringSlice("tags"),
		Monitoring: l.GetBool("monitoring"),
		IPv6:       l.GetBool("ipv6"),
		VPC:        parseVPCConfig(l.GetList("vpc")),
	}
}

func parseLinodeProvider(l *List) *LinodeProvider {
	return &LinodeProvider{
		Enabled:        l.GetBool("enabled"),
		Token:          l.GetString("token"),
		Region:         l.GetString("region"),
		RootPassword:   l.GetString("root-password"),
		PrivateIP:      l.GetBool("private-ip"),
		AuthorizedKeys: l.GetStringSlice("authorized-keys"),
		Tags:           l.GetStringSlice("tags"),
		VPC:            parseVPCConfig(l.GetList("vpc")),
	}
}

func parseAWSProvider(l *List) *AWSProvider {
	return &AWSProvider{
		Enabled:         l.GetBool("enabled"),
		AccessKeyID:     l.GetString("access-key-id"),
		SecretAccessKey: l.GetString("secret-access-key"),
		Region:          l.GetString("region"),
		SecurityGroups:  l.GetStringSlice("security-groups"),
		KeyPair:         l.GetString("key-pair"),
		IAMRole:         l.GetString("iam-role"),
		VPC:             parseVPCConfig(l.GetList("vpc")),
	}
}

func parseAzureProvider(l *List) *AzureProvider {
	return &AzureProvider{
		Enabled:        l.GetBool("enabled"),
		SubscriptionID: l.GetString("subscription-id"),
		TenantID:       l.GetString("tenant-id"),
		ClientID:       l.GetString("client-id"),
		ClientSecret:   l.GetString("client-secret"),
		ResourceGroup:  l.GetString("resource-group"),
		Location:       l.GetString("location"),
	}
}

func parseGCPProvider(l *List) *GCPProvider {
	return &GCPProvider{
		Enabled:     l.GetBool("enabled"),
		ProjectID:   l.GetString("project-id"),
		Credentials: l.GetString("credentials"),
		Region:      l.GetString("region"),
		Zone:        l.GetString("zone"),
	}
}

func parseHetznerProvider(l *List) *HetznerProvider {
	return &HetznerProvider{
		Enabled:    l.GetBool("enabled"),
		Token:      l.GetString("token"),
		Location:   l.GetString("location"),
		Datacenter: l.GetString("datacenter"),
		SSHKeys:    l.GetStringSlice("ssh-keys"),
	}
}

func parseVPCConfig(l *List) *VPCConfig {
	if l == nil {
		return nil
	}
	return &VPCConfig{
		Create:            l.GetBool("create"),
		ID:                l.GetString("id"),
		Name:              l.GetString("name"),
		CIDR:              l.GetString("cidr"),
		Region:            l.GetString("region"),
		Private:           l.GetBool("private"),
		EnableDNS:         l.GetBool("enable-dns"),
		EnableDNSHostname: l.GetBool("enable-dns-hostname"),
		InternetGateway:   l.GetBool("internet-gateway"),
		NATGateway:        l.GetBool("nat-gateway"),
	}
}

func parseNetwork(l *List) NetworkConfig {
	cfg := NetworkConfig{
		Mode:        l.GetString("mode"),
		CIDR:        l.GetString("cidr"),
		PodCIDR:     l.GetString("pod-cidr"),
		ServiceCIDR: l.GetString("service-cidr"),
		DNSServers:  l.GetStringSlice("dns-servers"),
	}

	if wg := l.GetList("wireguard"); wg != nil {
		cfg.WireGuard = parseWireGuard(wg)
	}

	if ts := l.GetList("tailscale"); ts != nil {
		cfg.Tailscale = parseTailscale(ts)
	}

	if dns := l.GetList("dns"); dns != nil {
		cfg.DNS = parseDNS(dns)
	}

	if fw := l.GetList("firewall"); fw != nil {
		cfg.Firewall = parseFirewall(fw)
	}

	if pc := l.GetList("private-cluster"); pc != nil {
		cfg.PrivateCluster = parsePrivateClusterConfig(pc)
	}

	return cfg
}

func parseWireGuard(l *List) *WireGuardConfig {
	return &WireGuardConfig{
		Enabled:             l.GetBool("enabled"),
		Create:              l.GetBool("create"),
		Provider:            l.GetString("provider"),
		Region:              l.GetString("region"),
		Size:                l.GetString("size"),
		Image:               l.GetString("image"),
		Name:                l.GetString("name"),
		ServerEndpoint:      l.GetString("server-endpoint"),
		ServerPublicKey:     l.GetString("server-public-key"),
		ClientIPBase:        l.GetString("client-ip-base"),
		Port:                l.GetInt("port"),
		MTU:                 l.GetInt("mtu"),
		PersistentKeepalive: l.GetInt("persistent-keepalive"),
		AutoConfig:          l.GetBool("auto-config"),
		MeshNetworking:      l.GetBool("mesh-networking"),
		SubnetCIDR:          l.GetString("subnet-cidr"),
	}
}

func parseTailscale(l *List) *TailscaleConfig {
	return &TailscaleConfig{
		Enabled:      l.GetBool("enabled"),
		HeadscaleURL: l.GetString("headscale-url"),
		APIKey:       l.GetString("api-key"),
		Namespace:    l.GetString("namespace"),
		AuthKey:      l.GetString("auth-key"),
		Tags:         l.GetStringSlice("tags"),
		AcceptRoutes: l.GetBool("accept-routes"),
		ExitNode:     l.GetString("exit-node"),
		Create:       l.GetBool("create"),
		Provider:     l.GetString("provider"),
		Region:       l.GetString("region"),
		Size:         l.GetString("size"),
		Domain:       l.GetString("domain"),
	}
}

func parseDNS(l *List) DNSConfig {
	return DNSConfig{
		Domain:      l.GetString("domain"),
		Servers:     l.GetStringSlice("servers"),
		Searches:    l.GetStringSlice("searches"),
		ExternalDNS: l.GetBool("external-dns"),
		Provider:    l.GetString("provider"),
	}
}

func parseFirewall(l *List) *FirewallConfig {
	cfg := &FirewallConfig{
		Name:          l.GetString("name"),
		DefaultAction: l.GetString("default-action"),
	}

	if inbound := l.GetList("inbound-rules"); inbound != nil {
		cfg.InboundRules = parseFirewallRules(inbound)
	}
	if outbound := l.GetList("outbound-rules"); outbound != nil {
		cfg.OutboundRules = parseFirewallRules(outbound)
	}

	return cfg
}

func parseFirewallRules(l *List) []FirewallRule {
	var rules []FirewallRule
	for _, item := range l.Items {
		if rule, ok := item.(*List); ok {
			rules = append(rules, FirewallRule{
				Protocol:    rule.GetString("protocol"),
				Port:        rule.GetString("port"),
				Source:      rule.GetStringSlice("source"),
				Target:      rule.GetStringSlice("target"),
				Action:      rule.GetString("action"),
				Description: rule.GetString("description"),
			})
		}
	}
	return rules
}

func parseSecurity(l *List) SecurityConfig {
	cfg := SecurityConfig{}

	if ssh := l.GetList("ssh"); ssh != nil {
		cfg.SSHConfig = parseSSHConfig(ssh)
	}

	if bastion := l.GetList("bastion"); bastion != nil {
		cfg.Bastion = parseBastionConfig(bastion)
	}

	return cfg
}

func parseSSHConfig(l *List) SSHConfig {
	return SSHConfig{
		AutoGenerate:      l.GetBool("auto-generate"),
		KeyPath:           l.GetString("key-path"),
		PublicKeyPath:     l.GetString("public-key-path"),
		AuthorizedKeys:    l.GetStringSlice("authorized-keys"),
		AllowPasswordAuth: l.GetBool("allow-password-auth"),
		Port:              l.GetInt("port"),
		AllowedUsers:      l.GetStringSlice("allowed-users"),
	}
}

func parseBastionConfig(l *List) *BastionConfig {
	return &BastionConfig{
		Enabled:        l.GetBool("enabled"),
		Provider:       l.GetString("provider"),
		Region:         l.GetString("region"),
		Size:           l.GetString("size"),
		Image:          l.GetString("image"),
		Name:           l.GetString("name"),
		VPNOnly:        l.GetBool("vpn-only"),
		AllowedCIDRs:   l.GetStringSlice("allowed-cidrs"),
		SSHPort:        l.GetInt("ssh-port"),
		IdleTimeout:    l.GetInt("idle-timeout"),
		MaxSessions:    l.GetInt("max-sessions"),
		EnableAuditLog: l.GetBool("enable-audit-log"),
		EnableMFA:      l.GetBool("enable-mfa"),
	}
}

func parseNodes(l *List) []NodeConfig {
	var nodes []NodeConfig
	for _, item := range l.Tail() {
		if node, ok := item.(*List); ok {
			nodes = append(nodes, NodeConfig{
				Name:         node.GetString("name"),
				Provider:     node.GetString("provider"),
				Pool:         node.GetString("pool"),
				Roles:        node.GetStringSlice("roles"),
				Size:         node.GetString("size"),
				Image:        node.GetString("image"),
				Region:       node.GetString("region"),
				Zone:         node.GetString("zone"),
				Labels:       node.GetMap("labels"),
				SpotInstance: node.GetBool("spot-instance"),
				SpotMaxPrice: node.GetString("spot-max-price"),
			})
		}
	}
	return nodes
}

func parseNodePools(l *List) map[string]NodePool {
	pools := make(map[string]NodePool)

	for _, item := range l.Tail() {
		if pool, ok := item.(*List); ok {
			if head := pool.Head(); head != nil {
				name := head.AsString()
				nodePool := NodePool{
					Name:         pool.GetString("name"),
					Provider:     pool.GetString("provider"),
					Count:        pool.GetInt("count"),
					MinCount:     pool.GetInt("min-count"),
					MaxCount:     pool.GetInt("max-count"),
					Roles:        pool.GetStringSlice("roles"),
					Size:         pool.GetString("size"),
					Image:        pool.GetString("image"),
					Region:       pool.GetString("region"),
					Zones:        pool.GetStringSlice("zones"),
					Labels:       pool.GetMap("labels"),
					AutoScaling:  pool.GetBool("auto-scaling"),
					SpotInstance: pool.GetBool("spot-instance"),
					Preemptible:  pool.GetBool("preemptible"),
					UserData:     pool.GetString("user-data"),
				}

				// Parse advanced configurations
				if autoscaling := pool.GetList("autoscaling"); autoscaling != nil {
					nodePool.AutoScalingConfig = parseAutoScalingConfig(autoscaling)
					nodePool.AutoScaling = nodePool.AutoScalingConfig.Enabled
				}

				if spotConfig := pool.GetList("spot-config"); spotConfig != nil {
					nodePool.SpotConfig = parseSpotConfig(spotConfig)
					nodePool.SpotInstance = nodePool.SpotConfig.Enabled
				}

				if distribution := pool.GetList("distribution"); distribution != nil {
					nodePool.Distribution = parseZoneDistribution(distribution)
				}

				if taints := pool.GetList("taints"); taints != nil {
					nodePool.Taints = parseTaints(taints)
				}

				if image := pool.GetList("image"); image != nil {
					nodePool.CustomImage = parseCustomImageConfig(image)
				}

				pools[name] = nodePool
			}
		}
	}

	return pools
}

func parseKubernetes(l *List) KubernetesConfig {
	cfg := KubernetesConfig{
		Version:       l.GetString("version"),
		Distribution:  l.GetString("distribution"),
		NetworkPlugin: l.GetString("network-plugin"),
		PodCIDR:       l.GetString("pod-cidr"),
		ServiceCIDR:   l.GetString("service-cidr"),
		ClusterDNS:    l.GetString("cluster-dns"),
		ClusterDomain: l.GetString("cluster-domain"),
	}

	if rke2 := l.GetList("rke2"); rke2 != nil {
		cfg.RKE2 = parseRKE2Config(rke2)
	}

	return cfg
}

func parseRKE2Config(l *List) *RKE2Config {
	return &RKE2Config{
		Version:              l.GetString("version"),
		Channel:              l.GetString("channel"),
		ClusterToken:         l.GetString("cluster-token"),
		TLSSan:               l.GetStringSlice("tls-san"),
		DisableComponents:    l.GetStringSlice("disable-components"),
		DataDir:              l.GetString("data-dir"),
		SnapshotScheduleCron: l.GetString("snapshot-schedule-cron"),
		SnapshotRetention:    l.GetInt("snapshot-retention"),
		SeLinux:              l.GetBool("selinux"),
		SecretsEncryption:    l.GetBool("secrets-encryption"),
	}
}

func parseMonitoring(l *List) MonitoringConfig {
	cfg := MonitoringConfig{
		Enabled:  l.GetBool("enabled"),
		Provider: l.GetString("provider"),
	}

	if prom := l.GetList("prometheus"); prom != nil {
		cfg.Prometheus = &PrometheusConfig{
			Enabled:        prom.GetBool("enabled"),
			Retention:      prom.GetString("retention"),
			StorageSize:    prom.GetString("storage-size"),
			Replicas:       prom.GetInt("replicas"),
			ScrapeInterval: prom.GetString("scrape-interval"),
		}
	}

	if grafana := l.GetList("grafana"); grafana != nil {
		cfg.Grafana = &GrafanaConfig{
			Enabled:       grafana.GetBool("enabled"),
			AdminPassword: grafana.GetString("admin-password"),
			Ingress:       grafana.GetBool("ingress"),
			Domain:        grafana.GetString("domain"),
		}
	}

	return cfg
}

func parseStorage(l *List) StorageConfig {
	return StorageConfig{
		DefaultClass: l.GetString("default-class"),
	}
}

func parseLoadBalancer(l *List) LoadBalancerConfig {
	return LoadBalancerConfig{
		Name:     l.GetString("name"),
		Type:     l.GetString("type"),
		Provider: l.GetString("provider"),
	}
}

func parseAddons(l *List) AddonsConfig {
	cfg := AddonsConfig{}

	if argocd := l.GetList("argocd"); argocd != nil {
		cfg.ArgoCD = &ArgoCDConfig{
			Enabled:          argocd.GetBool("enabled"),
			Version:          argocd.GetString("version"),
			GitOpsRepoURL:    argocd.GetString("gitops-repo-url"),
			GitOpsRepoBranch: argocd.GetString("gitops-repo-branch"),
			AppsPath:         argocd.GetString("apps-path"),
			Namespace:        argocd.GetString("namespace"),
		}
	}

	if salt := l.GetList("salt"); salt != nil {
		cfg.Salt = &SaltConfig{
			Enabled:      salt.GetBool("enabled"),
			MasterNode:   salt.GetString("master-node"), // Node name where Salt Master will be installed
			APIEnabled:   salt.GetBool("api-enabled"),
			APIPort:      salt.GetInt("api-port"),
			APIUsername:  salt.GetString("api-username"), // Salt API username (default: saltapi)
			APIPassword:  salt.GetString("api-password"), // Salt API password (auto-generated if empty)
			SecureAuth:   salt.GetBool("secure-auth"),
			AutoJoin:     salt.GetBool("auto-join"),
			AuditLogging: salt.GetBool("audit-logging"),
			StateRoots:   salt.GetString("state-roots"),
			PillarRoots:  salt.GetString("pillar-roots"),
			GitOpsRepo:   salt.GetString("gitops-repo"),
			GitOpsBranch: salt.GetString("gitops-branch"),
		}
		// Apply Salt defaults
		if cfg.Salt.APIPort == 0 {
			cfg.Salt.APIPort = 8000
		}
		// Default API username
		if cfg.Salt.APIUsername == "" {
			cfg.Salt.APIUsername = "saltapi"
		}
		// Note: APIPassword will be auto-generated in salt_master.go if empty
		// Default to secure auth if not specified
		if !salt.GetBool("secure-auth") && salt.Get("secure-auth") == nil {
			cfg.Salt.SecureAuth = true
		}
		// Default to auto-join if not specified
		if !salt.GetBool("auto-join") && salt.Get("auto-join") == nil {
			cfg.Salt.AutoJoin = true
		}
		// Default to API enabled
		if !salt.GetBool("api-enabled") && salt.Get("api-enabled") == nil {
			cfg.Salt.APIEnabled = true
		}
		// Default to audit logging
		if !salt.GetBool("audit-logging") && salt.Get("audit-logging") == nil {
			cfg.Salt.AuditLogging = true
		}
	}

	return cfg
}

// LoadConfig loads configuration from a Lisp file
func LoadConfig(filePath string) (*ClusterConfig, error) {
	return LoadFromLisp(filePath)
}

// parseUpgradeConfig parses upgrade configuration
func parseUpgradeConfig(l *List) *UpgradeConfig {
	return &UpgradeConfig{
		Strategy:            l.GetString("strategy"),
		MaxUnavailable:      l.GetInt("max-unavailable"),
		MaxSurge:            l.GetInt("max-surge"),
		DrainTimeout:        l.GetInt("drain-timeout"),
		HealthCheckInterval: l.GetInt("health-check-interval"),
		PauseOnFailure:      l.GetBool("pause-on-failure"),
		AutoRollback:        l.GetBool("auto-rollback"),
	}
}

// parseBackupConfig parses backup configuration
func parseBackupConfig(l *List) *BackupConfig {
	cfg := &BackupConfig{
		Enabled:        l.GetBool("enabled"),
		Schedule:       l.GetString("schedule"),
		Retention:      l.GetInt("retention"),
		RetentionDays:  l.GetInt("retention-days"),
		Provider:       l.GetString("provider"),
		Location:       l.GetString("location"),
		IncludeEtcd:    l.GetBool("include-etcd"),
		IncludeVolumes: l.GetBool("include-volumes"),
		Components:     l.GetStringSlice("components"),
	}

	if storage := l.GetList("storage"); storage != nil {
		cfg.Storage = parseBackupStorageConfig(storage)
	}

	return cfg
}

// parseBackupStorageConfig parses backup storage configuration
func parseBackupStorageConfig(l *List) *BackupStorageConfig {
	return &BackupStorageConfig{
		Type:      l.GetString("type"),
		Bucket:    l.GetString("bucket"),
		Region:    l.GetString("region"),
		Path:      l.GetString("path"),
		Endpoint:  l.GetString("endpoint"),
		AccessKey: l.GetString("access-key"),
		SecretKey: l.GetString("secret-key"),
	}
}

// parseHooksConfig parses provisioning hooks configuration
func parseHooksConfig(l *List) *HooksConfig {
	cfg := &HooksConfig{}

	if hooks := l.GetList("post-node-create"); hooks != nil {
		cfg.PostNodeCreate = parseHookActions(hooks)
	}
	if hooks := l.GetList("pre-cluster-destroy"); hooks != nil {
		cfg.PreClusterDestroy = parseHookActions(hooks)
	}
	if hooks := l.GetList("post-cluster-ready"); hooks != nil {
		cfg.PostClusterReady = parseHookActions(hooks)
	}
	if hooks := l.GetList("pre-node-delete"); hooks != nil {
		cfg.PreNodeDelete = parseHookActions(hooks)
	}
	if hooks := l.GetList("post-upgrade"); hooks != nil {
		cfg.PostUpgrade = parseHookActions(hooks)
	}

	return cfg
}

// parseHookActions parses a list of hook actions
func parseHookActions(l *List) []HookAction {
	var actions []HookAction
	for _, item := range l.Items {
		if action, ok := item.(*List); ok {
			head := action.Head()
			if head == nil {
				continue
			}
			hookAction := HookAction{
				Type:       head.AsString(),
				Command:    action.GetString("command"),
				Script:     action.GetString("script"),
				URL:        action.GetString("url"),
				Timeout:    action.GetInt("timeout"),
				RetryCount: action.GetInt("retry-count"),
				Env:        action.GetMap("env"),
			}
			// If it's a simple (script "path") or (kubectl "command") format
			if len(action.Items) == 2 {
				if val, ok := action.Items[1].(*Atom); ok {
					switch head.AsString() {
					case "script":
						hookAction.Script = val.AsString()
					case "kubectl":
						hookAction.Command = val.AsString()
						hookAction.Type = "kubectl"
					case "http":
						hookAction.URL = val.AsString()
					}
				}
			}
			actions = append(actions, hookAction)
		}
	}
	return actions
}

// parseCostControlConfig parses cost control configuration
func parseCostControlConfig(l *List) *CostControlConfig {
	return &CostControlConfig{
		Estimate:             l.GetBool("estimate"),
		MonthlyBudget:        float64(l.GetInt("monthly-limit")),
		AlertThreshold:       l.GetInt("alert-threshold"),
		NotifyEmail:          l.GetString("notify"),
		RightSizing:          l.GetBool("right-sizing"),
		UnusedResourcesAlert: l.GetBool("unused-resources-alert"),
		CostTags:             l.GetMap("cost-tags"),
	}
}

// parsePrivateClusterConfig parses private cluster configuration
func parsePrivateClusterConfig(l *List) *PrivateClusterConfig {
	return &PrivateClusterConfig{
		Enabled:         l.GetBool("enabled"),
		NATGateway:      l.GetBool("nat-gateway"),
		PrivateEndpoint: l.GetBool("private-endpoint"),
		PublicEndpoint:  l.GetBool("public-endpoint"),
		AllowedCIDRs:    l.GetStringSlice("allowed-cidrs"),
		VPNRequired:     l.GetBool("vpn-required"),
	}
}

// parseAutoScalingConfig parses autoscaling configuration for node pools
func parseAutoScalingConfig(l *List) *AutoScalingConfig {
	return &AutoScalingConfig{
		Enabled:        l.GetBool("enabled"),
		MinNodes:       l.GetInt("min-nodes"),
		MaxNodes:       l.GetInt("max-nodes"),
		TargetCPU:      l.GetInt("target-cpu"),
		TargetMemory:   l.GetInt("target-memory"),
		ScaleDownDelay: l.GetInt("scale-down-delay"),
		Cooldown:       l.GetInt("cooldown"),
	}
}

// parseSpotConfig parses spot instance configuration
func parseSpotConfig(l *List) *SpotConfig {
	return &SpotConfig{
		Enabled:          l.GetBool("enabled"),
		MaxPrice:         l.GetString("max-price"),
		FallbackOnDemand: l.GetBool("fallback-on-demand"),
		SpotPercentage:   l.GetInt("spot-percentage"),
		InterruptionMode: l.GetString("interruption-mode"),
	}
}

// parseZoneDistribution parses zone distribution configuration
func parseZoneDistribution(l *List) []ZoneDistribution {
	var distributions []ZoneDistribution
	for _, item := range l.Items {
		if zone, ok := item.(*List); ok {
			head := zone.Head()
			if head != nil && head.AsString() == "zone" {
				if len(zone.Items) >= 2 {
					dist := ZoneDistribution{
						Zone: zone.GetString("zone"),
					}
					// Parse (zone "us-east-1a" (count 2)) format
					if len(zone.Items) >= 2 {
						if zoneName, ok := zone.Items[1].(*Atom); ok {
							dist.Zone = zoneName.AsString()
						}
					}
					dist.Count = zone.GetInt("count")
					dist.Region = zone.GetString("region")
					distributions = append(distributions, dist)
				}
			}
		}
	}
	return distributions
}

// parseCustomImageConfig parses custom image configuration
func parseCustomImageConfig(l *List) *CustomImageConfig {
	return &CustomImageConfig{
		Type:         l.GetString("type"),
		ID:           l.GetString("id"),
		User:         l.GetString("user"),
		Base:         l.GetString("base"),
		Provisioners: l.GetStringSlice("provisioners"),
		Tags:         l.GetMap("tags"),
	}
}

// parseTaints parses taint configurations
func parseTaints(l *List) []TaintConfig {
	var taints []TaintConfig
	for _, item := range l.Items {
		if taint, ok := item.(*List); ok {
			head := taint.Head()
			if head != nil && head.AsString() == "taint" {
				// Parse (taint "key" "effect") or (taint (key "x") (value "y") (effect "z"))
				if len(taint.Items) == 3 {
					if key, ok := taint.Items[1].(*Atom); ok {
						if effect, ok := taint.Items[2].(*Atom); ok {
							taints = append(taints, TaintConfig{
								Key:    key.AsString(),
								Effect: effect.AsString(),
							})
						}
					}
				} else {
					taints = append(taints, TaintConfig{
						Key:    taint.GetString("key"),
						Value:  taint.GetString("value"),
						Effect: taint.GetString("effect"),
					})
				}
			}
		}
	}
	return taints
}
