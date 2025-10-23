// Package config provides comprehensive configuration types for the kubernetes-create cluster
package config

import (
	"time"
)

// ClusterConfig represents the complete cluster configuration
type ClusterConfig struct {
	Metadata     Metadata               `yaml:"metadata" json:"metadata"`
	Cluster      ClusterSpec            `yaml:"cluster" json:"cluster"`
	Providers    ProvidersConfig        `yaml:"providers" json:"providers"`
	Network      NetworkConfig          `yaml:"network" json:"network"`
	Security     SecurityConfig         `yaml:"security" json:"security"`
	Nodes        []NodeConfig           `yaml:"nodes" json:"nodes"`
	NodePools    map[string]NodePool    `yaml:"nodePools" json:"nodePools"`
	Kubernetes   KubernetesConfig       `yaml:"kubernetes" json:"kubernetes"`
	Monitoring   MonitoringConfig       `yaml:"monitoring" json:"monitoring"`
	Storage      StorageConfig          `yaml:"storage" json:"storage"`
	LoadBalancer LoadBalancerConfig     `yaml:"loadBalancer" json:"loadBalancer"`
	Addons       AddonsConfig           `yaml:"addons" json:"addons"`
}

// AddonsConfig defines cluster addons configuration
type AddonsConfig struct {
	ArgoCD *ArgoCDConfig `yaml:"argocd,omitempty" json:"argocd,omitempty"`
}

// ArgoCDConfig defines ArgoCD GitOps configuration
type ArgoCDConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Version        string `yaml:"version" json:"version"` // ArgoCD version to install
	GitOpsRepoURL  string `yaml:"gitopsRepoUrl" json:"gitopsRepoUrl"`
	GitOpsRepoBranch string `yaml:"gitopsRepoBranch" json:"gitopsRepoBranch"` // default: main
	AppsPath       string `yaml:"appsPath" json:"appsPath"` // default: argocd/apps
	Namespace      string `yaml:"namespace" json:"namespace"` // default: argocd
	AdminPassword  string `yaml:"adminPassword,omitempty" json:"adminPassword,omitempty"`
}

// Metadata contains cluster metadata
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Environment string            `yaml:"environment" json:"environment"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	Owner       string            `yaml:"owner" json:"owner"`
	Team        string            `yaml:"team" json:"team"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

// ClusterSpec defines the cluster specifications
type ClusterSpec struct {
	Type              string            `yaml:"type" json:"type"` // rke, k3s, eks, gke, aks
	Version           string            `yaml:"version" json:"version"`
	Distribution      string            `yaml:"distribution" json:"distribution"`
	HighAvailability  bool              `yaml:"highAvailability" json:"highAvailability"`
	MultiCloud        bool              `yaml:"multiCloud" json:"multiCloud"`
	AutoScaling       AutoScalingConfig `yaml:"autoScaling" json:"autoScaling"`
	BackupConfig      BackupConfig      `yaml:"backup" json:"backup"`
	MaintenanceWindow MaintenanceWindow `yaml:"maintenanceWindow" json:"maintenanceWindow"`
}

// ProvidersConfig configures cloud providers
type ProvidersConfig struct {
	DigitalOcean *DigitalOceanProvider `yaml:"digitalocean,omitempty" json:"digitalocean,omitempty"`
	Linode       *LinodeProvider       `yaml:"linode,omitempty" json:"linode,omitempty"`
	AWS          *AWSProvider          `yaml:"aws,omitempty" json:"aws,omitempty"`
	Azure        *AzureProvider        `yaml:"azure,omitempty" json:"azure,omitempty"`
	GCP          *GCPProvider          `yaml:"gcp,omitempty" json:"gcp,omitempty"`
}

// DigitalOceanProvider configuration
type DigitalOceanProvider struct {
	Enabled      bool                   `yaml:"enabled" json:"enabled"`
	Token        string                 `yaml:"token" json:"token"`
	Region       string                 `yaml:"region" json:"region"`
	VPC          *VPCConfig             `yaml:"vpc,omitempty" json:"vpc,omitempty"`
	SSHKeys      []string               `yaml:"sshKeys" json:"sshKeys"`
	SSHPublicKey interface{}            `yaml:"-" json:"-"` // Set programmatically
	Tags         []string               `yaml:"tags" json:"tags"`
	Monitoring   bool                   `yaml:"monitoring" json:"monitoring"`
	IPv6         bool                   `yaml:"ipv6" json:"ipv6"`
	UserData     string                 `yaml:"userData" json:"userData"`
	BackupPolicy *BackupPolicy          `yaml:"backupPolicy,omitempty" json:"backupPolicy,omitempty"`
	Firewall     *FirewallConfig        `yaml:"firewall,omitempty" json:"firewall,omitempty"`
	Custom       map[string]interface{} `yaml:"custom" json:"custom"`
}

// LinodeProvider configuration
type LinodeProvider struct {
	Enabled        bool                   `yaml:"enabled" json:"enabled"`
	Token          string                 `yaml:"token" json:"token"`
	Region         string                 `yaml:"region" json:"region"`
	RootPassword   string                 `yaml:"rootPassword" json:"rootPassword"`
	PrivateIP      bool                   `yaml:"privateIp" json:"privateIp"`
	AuthorizedKeys []string               `yaml:"authorizedKeys" json:"authorizedKeys"`
	SSHPublicKey   interface{}            `yaml:"-" json:"-"` // Set programmatically
	Tags           []string               `yaml:"tags" json:"tags"`
	VPC            *VPCConfig             `yaml:"vpc,omitempty" json:"vpc,omitempty"`
	BackupPolicy   *BackupPolicy          `yaml:"backupPolicy,omitempty" json:"backupPolicy,omitempty"`
	Firewall       *FirewallConfig        `yaml:"firewall,omitempty" json:"firewall,omitempty"`
	Custom         map[string]interface{} `yaml:"custom" json:"custom"`
}

// AWSProvider configuration
type AWSProvider struct {
	Enabled         bool                   `yaml:"enabled" json:"enabled"`
	AccessKeyID     string                 `yaml:"accessKeyId" json:"accessKeyId"`
	SecretAccessKey string                 `yaml:"secretAccessKey" json:"secretAccessKey"`
	Region          string                 `yaml:"region" json:"region"`
	VPC             *VPCConfig             `yaml:"vpc,omitempty" json:"vpc,omitempty"`
	SecurityGroups  []string               `yaml:"securityGroups" json:"securityGroups"`
	KeyPair         string                 `yaml:"keyPair" json:"keyPair"`
	IAMRole         string                 `yaml:"iamRole" json:"iamRole"`
	Custom          map[string]interface{} `yaml:"custom" json:"custom"`
}

// AzureProvider configuration
type AzureProvider struct {
	Enabled        bool                   `yaml:"enabled" json:"enabled"`
	SubscriptionID string                 `yaml:"subscriptionId" json:"subscriptionId"`
	TenantID       string                 `yaml:"tenantId" json:"tenantId"`
	ClientID       string                 `yaml:"clientId" json:"clientId"`
	ClientSecret   string                 `yaml:"clientSecret" json:"clientSecret"`
	ResourceGroup  string                 `yaml:"resourceGroup" json:"resourceGroup"`
	Location       string                 `yaml:"location" json:"location"`
	VirtualNetwork *VPCConfig             `yaml:"virtualNetwork,omitempty" json:"virtualNetwork,omitempty"`
	Custom         map[string]interface{} `yaml:"custom" json:"custom"`
}

// GCPProvider configuration
type GCPProvider struct {
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	ProjectID   string                 `yaml:"projectId" json:"projectId"`
	Credentials string                 `yaml:"credentials" json:"credentials"`
	Region      string                 `yaml:"region" json:"region"`
	Zone        string                 `yaml:"zone" json:"zone"`
	Network     *VPCConfig             `yaml:"network,omitempty" json:"network,omitempty"`
	Custom      map[string]interface{} `yaml:"custom" json:"custom"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	Mode                    string                 `yaml:"mode" json:"mode"` // vpc, wireguard, tailscale, hybrid
	CIDR                    string                 `yaml:"cidr" json:"cidr"`
	PodCIDR                 string                 `yaml:"podCidr" json:"podCidr"`
	ServiceCIDR             string                 `yaml:"serviceCidr" json:"serviceCidr"`
	Subnets                 []SubnetConfig         `yaml:"subnets" json:"subnets"`
	DNS                     DNSConfig              `yaml:"dns" json:"dns"`
	DNSServers              []string               `yaml:"dnsServers" json:"dnsServers"`
	EnableNodePorts         bool                   `yaml:"enableNodePorts" json:"enableNodePorts"`
	CrossProviderNetworking bool                   `yaml:"crossProviderNetworking" json:"crossProviderNetworking"`
	LoadBalancers           []LoadBalancerConfig   `yaml:"loadBalancers" json:"loadBalancers"`
	Ingress                 IngressConfig          `yaml:"ingress" json:"ingress"`
	ServiceMesh             *ServiceMeshConfig     `yaml:"serviceMesh,omitempty" json:"serviceMesh,omitempty"`
	NetworkPolicies         []NetworkPolicy        `yaml:"networkPolicies" json:"networkPolicies"`
	WireGuard               *WireGuardConfig       `yaml:"wireguard,omitempty" json:"wireguard,omitempty"`
	Firewall                *FirewallConfig        `yaml:"firewall,omitempty" json:"firewall,omitempty"`
	Custom                  map[string]interface{} `yaml:"custom" json:"custom"`
}

// WireGuardConfig for VPN setup
type WireGuardConfig struct {
	// Creation settings
	Create          bool   `yaml:"create" json:"create"`                   // Auto-create WireGuard server
	Provider        string `yaml:"provider" json:"provider"`               // Provider for WG server (digitalocean/linode)
	Region          string `yaml:"region" json:"region"`                   // Region for WG server
	Size            string `yaml:"size" json:"size"`                       // Server size
	Image           string `yaml:"image" json:"image"`                     // Server image (default: ubuntu-22-04-x64)
	Name            string `yaml:"name" json:"name"`                       // Server name
	ServerIPAddress string `yaml:"serverIpAddress" json:"serverIpAddress"` // Server public IP (auto-set if creating)

	// Connection settings (used if Create=false, or auto-generated if Create=true)
	Enabled             bool            `yaml:"enabled" json:"enabled"`
	ServerEndpoint      string          `yaml:"serverEndpoint" json:"serverEndpoint"`
	ServerPublicKey     string          `yaml:"serverPublicKey" json:"serverPublicKey"`
	ServerPrivateKey    string          `yaml:"serverPrivateKey" json:"serverPrivateKey"` // Only if creating
	ClientIPBase        string          `yaml:"clientIpBase" json:"clientIpBase"`
	Port                int             `yaml:"port" json:"port"`
	AllowedIPs          []string        `yaml:"allowedIps" json:"allowedIps"`
	DNS                 []string        `yaml:"dns" json:"dns"`
	MTU                 int             `yaml:"mtu" json:"mtu"`
	PersistentKeepalive int             `yaml:"persistentKeepalive" json:"persistentKeepalive"`
	Peers               []WireGuardPeer `yaml:"peers" json:"peers"`
	AutoConfig          bool            `yaml:"autoConfig" json:"autoConfig"`
	MeshNetworking      bool            `yaml:"meshNetworking" json:"meshNetworking"`
	SSHPrivateKeyPath   string          `yaml:"sshPrivateKeyPath" json:"sshPrivateKeyPath"`

	// Network configuration
	SubnetCIDR string `yaml:"subnetCidr" json:"subnetCidr"` // VPN subnet (e.g., 10.8.0.0/24)
}

// WireGuardPeer represents a WireGuard peer
type WireGuardPeer struct {
	Name         string   `yaml:"name" json:"name"`
	PublicKey    string   `yaml:"publicKey" json:"publicKey"`
	AllowedIPs   []string `yaml:"allowedIps" json:"allowedIps"`
	Endpoint     string   `yaml:"endpoint" json:"endpoint"`
	PresharedKey string   `yaml:"presharedKey" json:"presharedKey"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	SSHConfig       SSHConfig              `yaml:"ssh" json:"ssh"`
	Bastion         *BastionConfig         `yaml:"bastion,omitempty" json:"bastion,omitempty"`
	TLS             TLSConfig              `yaml:"tls" json:"tls"`
	RBAC            RBACConfig             `yaml:"rbac" json:"rbac"`
	PodSecurity     PodSecurityConfig      `yaml:"podSecurity" json:"podSecurity"`
	NetworkPolicies bool                   `yaml:"networkPolicies" json:"networkPolicies"`
	Secrets         SecretsConfig          `yaml:"secrets" json:"secrets"`
	Compliance      ComplianceConfig       `yaml:"compliance" json:"compliance"`
	Audit           AuditConfig            `yaml:"audit" json:"audit"`
	Custom          map[string]interface{} `yaml:"custom" json:"custom"`
}

// BastionConfig defines bastion host configuration for secure cluster access
type BastionConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	Provider       string   `yaml:"provider" json:"provider"` // digitalocean, linode, aws, gcp, azure
	Region         string   `yaml:"region" json:"region"`
	Size           string   `yaml:"size" json:"size"`
	Image          string   `yaml:"image" json:"image"`
	Name           string   `yaml:"name" json:"name"`
	VPNOnly        bool     `yaml:"vpnOnly" json:"vpnOnly"`               // If true, only VPN users can SSH to bastion
	AllowedCIDRs   []string `yaml:"allowedCIDRs" json:"allowedCIDRs"`     // CIDRs allowed to SSH to bastion
	SSHPort        int      `yaml:"sshPort" json:"sshPort"`               // Custom SSH port (default: 22)
	IdleTimeout    int      `yaml:"idleTimeout" json:"idleTimeout"`       // SSH idle timeout in minutes
	MaxSessions    int      `yaml:"maxSessions" json:"maxSessions"`       // Max concurrent SSH sessions
	EnableAuditLog bool     `yaml:"enableAuditLog" json:"enableAuditLog"` // Log all SSH sessions
	EnableMFA      bool     `yaml:"enableMFA" json:"enableMFA"`           // Require MFA for bastion access
}

// NodeConfig represents individual node configuration
type NodeConfig struct {
	Name        string                 `yaml:"name" json:"name"`
	Provider    string                 `yaml:"provider" json:"provider"`
	Pool        string                 `yaml:"pool" json:"pool"`
	Roles       []string               `yaml:"roles" json:"roles"`
	Size        string                 `yaml:"size" json:"size"`
	Image       string                 `yaml:"image" json:"image"`
	Region      string                 `yaml:"region" json:"region"`
	Zone        string                 `yaml:"zone" json:"zone"`
	PrivateIP   string                 `yaml:"privateIp" json:"privateIp"`
	PublicIP    string                 `yaml:"publicIp" json:"publicIp"`
	WireGuardIP string                 `yaml:"wireguardIp" json:"wireguardIp"`
	Labels      map[string]string      `yaml:"labels" json:"labels"`
	Taints      []TaintConfig          `yaml:"taints" json:"taints"`
	UserData    string                 `yaml:"userData" json:"userData"`
	SSHKey      string                 `yaml:"sshKey" json:"sshKey"`
	Monitoring  bool                   `yaml:"monitoring" json:"monitoring"`
	Custom      map[string]interface{} `yaml:"custom" json:"custom"`
}

// NodePool defines a pool of similar nodes
type NodePool struct {
	Name         string                 `yaml:"name" json:"name"`
	Provider     string                 `yaml:"provider" json:"provider"`
	Count        int                    `yaml:"count" json:"count"`
	MinCount     int                    `yaml:"minCount" json:"minCount"`
	MaxCount     int                    `yaml:"maxCount" json:"maxCount"`
	Roles        []string               `yaml:"roles" json:"roles"`
	Size         string                 `yaml:"size" json:"size"`
	Image        string                 `yaml:"image" json:"image"`
	Region       string                 `yaml:"region" json:"region"`
	Zones        []string               `yaml:"zones" json:"zones"`
	Labels       map[string]string      `yaml:"labels" json:"labels"`
	Taints       []TaintConfig          `yaml:"taints" json:"taints"`
	AutoScaling  bool                   `yaml:"autoScaling" json:"autoScaling"`
	SpotInstance bool                   `yaml:"spotInstance" json:"spotInstance"`
	Preemptible  bool                   `yaml:"preemptible" json:"preemptible"`
	UserData     string                 `yaml:"userData" json:"userData"`
	Custom       map[string]interface{} `yaml:"custom" json:"custom"`
}

// KubernetesConfig for Kubernetes-specific settings
type KubernetesConfig struct {
	Version           string                 `yaml:"version" json:"version"`
	Distribution      string                 `yaml:"distribution" json:"distribution"` // rke2, k3s, kubeadm
	NetworkPlugin     string                 `yaml:"networkPlugin" json:"networkPlugin"`
	PodCIDR           string                 `yaml:"podCidr" json:"podCidr"`
	ServiceCIDR       string                 `yaml:"serviceCidr" json:"serviceCidr"`
	ClusterDNS        string                 `yaml:"clusterDns" json:"clusterDns"`
	ClusterDomain     string                 `yaml:"clusterDomain" json:"clusterDomain"`
	RKE2              *RKE2Config            `yaml:"rke2,omitempty" json:"rke2,omitempty"`
	APIServer         APIServerConfig        `yaml:"apiServer" json:"apiServer"`
	ControllerManager ControllerConfig       `yaml:"controllerManager" json:"controllerManager"`
	Scheduler         SchedulerConfig        `yaml:"scheduler" json:"scheduler"`
	Kubelet           KubeletConfig          `yaml:"kubelet" json:"kubelet"`
	Etcd              EtcdConfig             `yaml:"etcd" json:"etcd"`
	Addons            []AddonConfig          `yaml:"addons" json:"addons"`
	Features          map[string]bool        `yaml:"features" json:"features"`
	Admission         AdmissionConfig        `yaml:"admission" json:"admission"`
	AuditLog          bool                   `yaml:"auditLog" json:"auditLog"`
	EncryptSecrets    bool                   `yaml:"encryptSecrets" json:"encryptSecrets"`
	Monitoring        bool                   `yaml:"monitoring" json:"monitoring"`
	Custom            map[string]interface{} `yaml:"custom" json:"custom"`
}

// RKE2Config specific configuration for RKE2 distribution
type RKE2Config struct {
	Version                  string            `yaml:"version" json:"version"`                                   // e.g., "v1.28.5+rke2r1"
	Channel                  string            `yaml:"channel" json:"channel"`                                   // stable, latest, testing
	ClusterToken             string            `yaml:"clusterToken" json:"clusterToken"`                         // Shared secret for cluster
	TLSSan                   []string          `yaml:"tlsSan" json:"tlsSan"`                                     // Additional SANs for API server
	DisableComponents        []string          `yaml:"disableComponents" json:"disableComponents"`               // Components to disable (e.g., rke2-ingress-nginx)
	DataDir                  string            `yaml:"dataDir" json:"dataDir"`                                   // Data directory (default: /var/lib/rancher/rke2)
	NodeTaint                []string          `yaml:"nodeTaint" json:"nodeTaint"`                               // Taints to apply to nodes
	NodeLabel                []string          `yaml:"nodeLabel" json:"nodeLabel"`                               // Labels to apply to nodes
	ContainerRuntimeEndpoint string            `yaml:"containerRuntimeEndpoint" json:"containerRuntimeEndpoint"` // Container runtime endpoint
	SnapshotScheduleCron     string            `yaml:"snapshotScheduleCron" json:"snapshotScheduleCron"`         // Etcd snapshot schedule
	SnapshotRetention        int               `yaml:"snapshotRetention" json:"snapshotRetention"`               // Number of snapshots to retain
	SystemDefaultRegistry    string            `yaml:"systemDefaultRegistry" json:"systemDefaultRegistry"`       // Private registry for system images
	Profiles                 []string          `yaml:"profiles" json:"profiles"`                                 // CIS profiles to enable
	SeLinux                  bool              `yaml:"selinux" json:"selinux"`                                   // Enable SELinux
	SecretsEncryption        bool              `yaml:"secretsEncryption" json:"secretsEncryption"`               // Enable secrets encryption
	WriteKubeconfigMode      string            `yaml:"writeKubeconfigMode" json:"writeKubeconfigMode"`           // Kubeconfig file permissions
	ProtectKernelDefaults    bool              `yaml:"protectKernelDefaults" json:"protectKernelDefaults"`       // Protect kernel defaults
	ExtraServerArgs          map[string]string `yaml:"extraServerArgs" json:"extraServerArgs"`                   // Extra arguments for server
	ExtraAgentArgs           map[string]string `yaml:"extraAgentArgs" json:"extraAgentArgs"`                     // Extra arguments for agent
}

// Helper types for various configurations
type VPCConfig struct {
	// Creation settings
	Create  bool   `yaml:"create" json:"create"`   // Auto-create VPC
	ID      string `yaml:"id" json:"id"`           // Existing VPC ID (if not creating)
	Name    string `yaml:"name" json:"name"`       // VPC name
	CIDR    string `yaml:"cidr" json:"cidr"`       // VPC CIDR block
	Region  string `yaml:"region" json:"region"`   // VPC region
	Private bool   `yaml:"private" json:"private"` // Private VPC

	// Advanced settings
	EnableDNS         bool     `yaml:"enableDns" json:"enableDns"`                 // Enable DNS resolution
	EnableDNSHostname bool     `yaml:"enableDnsHostname" json:"enableDnsHostname"` // Enable DNS hostnames
	Subnets           []string `yaml:"subnets" json:"subnets"`                     // Subnet CIDRs to create
	InternetGateway   bool     `yaml:"internetGateway" json:"internetGateway"`     // Create internet gateway
	NATGateway        bool     `yaml:"natGateway" json:"natGateway"`               // Create NAT gateway
	Tags              []string `yaml:"tags" json:"tags"`                           // VPC tags

	// Provider-specific
	DigitalOcean *DOVPCConfig     `yaml:"digitalocean,omitempty" json:"digitalocean,omitempty"`
	Linode       *LinodeVPCConfig `yaml:"linode,omitempty" json:"linode,omitempty"`
}

type SubnetConfig struct {
	Name             string   `yaml:"name" json:"name"`
	CIDR             string   `yaml:"cidr" json:"cidr"`
	Zone             string   `yaml:"zone" json:"zone"`
	Public           bool     `yaml:"public" json:"public"`
	RouteTable       string   `yaml:"routeTable" json:"routeTable"`
	NATGateway       bool     `yaml:"natGateway" json:"natGateway"`
	AvailabilityZone string   `yaml:"availabilityZone" json:"availabilityZone"`
	Tags             []string `yaml:"tags" json:"tags"`
}

type FirewallConfig struct {
	Name          string         `yaml:"name" json:"name"`
	InboundRules  []FirewallRule `yaml:"inboundRules" json:"inboundRules"`
	OutboundRules []FirewallRule `yaml:"outboundRules" json:"outboundRules"`
	DefaultAction string         `yaml:"defaultAction" json:"defaultAction"`
}

type FirewallRule struct {
	Protocol    string   `yaml:"protocol" json:"protocol"`
	Port        string   `yaml:"port" json:"port"`
	Source      []string `yaml:"source" json:"source"`
	Target      []string `yaml:"target" json:"target"`
	Action      string   `yaml:"action" json:"action"`
	Description string   `yaml:"description" json:"description"`
}

type DNSConfig struct {
	Domain      string   `yaml:"domain" json:"domain"`
	Servers     []string `yaml:"servers" json:"servers"`
	Searches    []string `yaml:"searches" json:"searches"`
	Options     []string `yaml:"options" json:"options"`
	ExternalDNS bool     `yaml:"externalDns" json:"externalDns"`
	Provider    string   `yaml:"provider" json:"provider"` // digitalocean, cloudflare, route53, etc
}

type IngressConfig struct {
	Controller string                 `yaml:"controller" json:"controller"`
	Class      string                 `yaml:"class" json:"class"`
	TLS        bool                   `yaml:"tls" json:"tls"`
	Replicas   int                    `yaml:"replicas" json:"replicas"`
	Custom     map[string]interface{} `yaml:"custom" json:"custom"`
}

type LoadBalancerConfig struct {
	Name     string                 `yaml:"name" json:"name"`
	Type     string                 `yaml:"type" json:"type"`
	Provider string                 `yaml:"provider" json:"provider"`
	Ports    []PortConfig           `yaml:"ports" json:"ports"`
	Custom   map[string]interface{} `yaml:"custom" json:"custom"`
}

type PortConfig struct {
	Name       string `yaml:"name" json:"name"`
	Port       int    `yaml:"port" json:"port"`
	TargetPort int    `yaml:"targetPort" json:"targetPort"`
	Protocol   string `yaml:"protocol" json:"protocol"`
}

type ServiceMeshConfig struct {
	Type    string                 `yaml:"type" json:"type"` // istio, linkerd, consul
	Version string                 `yaml:"version" json:"version"`
	MTLS    bool                   `yaml:"mtls" json:"mtls"`
	Tracing bool                   `yaml:"tracing" json:"tracing"`
	Custom  map[string]interface{} `yaml:"custom" json:"custom"`
}

type NetworkPolicy struct {
	Name      string                 `yaml:"name" json:"name"`
	Namespace string                 `yaml:"namespace" json:"namespace"`
	Rules     []PolicyRule           `yaml:"rules" json:"rules"`
	Custom    map[string]interface{} `yaml:"custom" json:"custom"`
}

type PolicyRule struct {
	Direction string   `yaml:"direction" json:"direction"`
	Protocol  string   `yaml:"protocol" json:"protocol"`
	Port      int      `yaml:"port" json:"port"`
	From      []string `yaml:"from" json:"from"`
	To        []string `yaml:"to" json:"to"`
}

type SSHConfig struct {
	KeyPath           string   `yaml:"keyPath" json:"keyPath"`
	PublicKeyPath     string   `yaml:"publicKeyPath" json:"publicKeyPath"`
	AuthorizedKeys    []string `yaml:"authorizedKeys" json:"authorizedKeys"`
	AllowPasswordAuth bool     `yaml:"allowPasswordAuth" json:"allowPasswordAuth"`
	Port              int      `yaml:"port" json:"port"`
	AllowedUsers      []string `yaml:"allowedUsers" json:"allowedUsers"`
}

type TLSConfig struct {
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	CertManager bool     `yaml:"certManager" json:"certManager"`
	Provider    string   `yaml:"provider" json:"provider"`
	Email       string   `yaml:"email" json:"email"`
	Domains     []string `yaml:"domains" json:"domains"`
}

type RBACConfig struct {
	Enabled       bool            `yaml:"enabled" json:"enabled"`
	DefaultPolicy string          `yaml:"defaultPolicy" json:"defaultPolicy"`
	Roles         []RoleConfig    `yaml:"roles" json:"roles"`
	Bindings      []BindingConfig `yaml:"bindings" json:"bindings"`
}

type RoleConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Namespace   string   `yaml:"namespace" json:"namespace"`
	Rules       []string `yaml:"rules" json:"rules"`
	ClusterRole bool     `yaml:"clusterRole" json:"clusterRole"`
}

type BindingConfig struct {
	Name      string   `yaml:"name" json:"name"`
	Role      string   `yaml:"role" json:"role"`
	Subjects  []string `yaml:"subjects" json:"subjects"`
	Namespace string   `yaml:"namespace" json:"namespace"`
}

type PodSecurityConfig struct {
	PolicyLevel    string `yaml:"policyLevel" json:"policyLevel"`
	EnforceProfile string `yaml:"enforceProfile" json:"enforceProfile"`
	AuditProfile   string `yaml:"auditProfile" json:"auditProfile"`
	WarnProfile    string `yaml:"warnProfile" json:"warnProfile"`
}

type SecretsConfig struct {
	Provider        string                 `yaml:"provider" json:"provider"`
	Encryption      bool                   `yaml:"encryption" json:"encryption"`
	KeyManagement   string                 `yaml:"keyManagement" json:"keyManagement"`
	ExternalSecrets bool                   `yaml:"externalSecrets" json:"externalSecrets"`
	Custom          map[string]interface{} `yaml:"custom" json:"custom"`
}

type ComplianceConfig struct {
	Standards []string               `yaml:"standards" json:"standards"`
	Scanning  bool                   `yaml:"scanning" json:"scanning"`
	Reporting bool                   `yaml:"reporting" json:"reporting"`
	Custom    map[string]interface{} `yaml:"custom" json:"custom"`
}

type AuditConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Level    string   `yaml:"level" json:"level"`
	Backend  string   `yaml:"backend" json:"backend"`
	Rotation string   `yaml:"rotation" json:"rotation"`
	Filters  []string `yaml:"filters" json:"filters"`
}

type TaintConfig struct {
	Key    string `yaml:"key" json:"key"`
	Value  string `yaml:"value" json:"value"`
	Effect string `yaml:"effect" json:"effect"`
}

type AutoScalingConfig struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	MinNodes     int    `yaml:"minNodes" json:"minNodes"`
	MaxNodes     int    `yaml:"maxNodes" json:"maxNodes"`
	TargetCPU    int    `yaml:"targetCpu" json:"targetCpu"`
	TargetMemory int    `yaml:"targetMemory" json:"targetMemory"`
	ScaleDown    string `yaml:"scaleDown" json:"scaleDown"`
	ScaleUp      string `yaml:"scaleUp" json:"scaleUp"`
}

type BackupConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Schedule       string `yaml:"schedule" json:"schedule"`
	Retention      int    `yaml:"retention" json:"retention"`
	Provider       string `yaml:"provider" json:"provider"`
	Location       string `yaml:"location" json:"location"`
	IncludeEtcd    bool   `yaml:"includeEtcd" json:"includeEtcd"`
	IncludeVolumes bool   `yaml:"includeVolumes" json:"includeVolumes"`
}

type BackupPolicy struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Frequency string `yaml:"frequency" json:"frequency"`
	Retention int    `yaml:"retention" json:"retention"`
}

type MaintenanceWindow struct {
	Day       string `yaml:"day" json:"day"`
	StartTime string `yaml:"startTime" json:"startTime"`
	Duration  string `yaml:"duration" json:"duration"`
	TimeZone  string `yaml:"timeZone" json:"timeZone"`
}

type MonitoringConfig struct {
	Enabled      bool                   `yaml:"enabled" json:"enabled"`
	Provider     string                 `yaml:"provider" json:"provider"`
	Prometheus   *PrometheusConfig      `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Grafana      *GrafanaConfig         `yaml:"grafana,omitempty" json:"grafana,omitempty"`
	AlertManager *AlertManagerConfig    `yaml:"alertManager,omitempty" json:"alertManager,omitempty"`
	Logging      *LoggingConfig         `yaml:"logging,omitempty" json:"logging,omitempty"`
	Tracing      *TracingConfig         `yaml:"tracing,omitempty" json:"tracing,omitempty"`
	Custom       map[string]interface{} `yaml:"custom" json:"custom"`
}

type PrometheusConfig struct {
	Enabled        bool              `yaml:"enabled" json:"enabled"`
	Retention      string            `yaml:"retention" json:"retention"`
	StorageSize    string            `yaml:"storageSize" json:"storageSize"`
	Replicas       int               `yaml:"replicas" json:"replicas"`
	ScrapeInterval string            `yaml:"scrapeInterval" json:"scrapeInterval"`
	ExternalLabels map[string]string `yaml:"externalLabels" json:"externalLabels"`
}

type GrafanaConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	AdminPassword string   `yaml:"adminPassword" json:"adminPassword"`
	Ingress       bool     `yaml:"ingress" json:"ingress"`
	Domain        string   `yaml:"domain" json:"domain"`
	Dashboards    []string `yaml:"dashboards" json:"dashboards"`
}

type AlertManagerConfig struct {
	Enabled   bool            `yaml:"enabled" json:"enabled"`
	Replicas  int             `yaml:"replicas" json:"replicas"`
	Routes    []AlertRoute    `yaml:"routes" json:"routes"`
	Receivers []AlertReceiver `yaml:"receivers" json:"receivers"`
}

type AlertRoute struct {
	Match    map[string]string `yaml:"match" json:"match"`
	Receiver string            `yaml:"receiver" json:"receiver"`
}

type AlertReceiver struct {
	Name   string                 `yaml:"name" json:"name"`
	Type   string                 `yaml:"type" json:"type"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

type LoggingConfig struct {
	Provider    string   `yaml:"provider" json:"provider"`
	Backend     string   `yaml:"backend" json:"backend"`
	Retention   string   `yaml:"retention" json:"retention"`
	Aggregation bool     `yaml:"aggregation" json:"aggregation"`
	Parsers     []string `yaml:"parsers" json:"parsers"`
}

type TracingConfig struct {
	Provider string  `yaml:"provider" json:"provider"`
	Endpoint string  `yaml:"endpoint" json:"endpoint"`
	Sampling float64 `yaml:"sampling" json:"sampling"`
}

type StorageConfig struct {
	DefaultClass      string                 `yaml:"defaultClass" json:"defaultClass"`
	Classes           []StorageClass         `yaml:"classes" json:"classes"`
	PersistentVolumes []PersistentVolume     `yaml:"persistentVolumes" json:"persistentVolumes"`
	CSIDrivers        []CSIDriver            `yaml:"csiDrivers" json:"csiDrivers"`
	Custom            map[string]interface{} `yaml:"custom" json:"custom"`
}

type StorageClass struct {
	Name              string            `yaml:"name" json:"name"`
	Provisioner       string            `yaml:"provisioner" json:"provisioner"`
	ReclaimPolicy     string            `yaml:"reclaimPolicy" json:"reclaimPolicy"`
	VolumeBindingMode string            `yaml:"volumeBindingMode" json:"volumeBindingMode"`
	Parameters        map[string]string `yaml:"parameters" json:"parameters"`
}

type PersistentVolume struct {
	Name         string            `yaml:"name" json:"name"`
	Size         string            `yaml:"size" json:"size"`
	StorageClass string            `yaml:"storageClass" json:"storageClass"`
	AccessModes  []string          `yaml:"accessModes" json:"accessModes"`
	Labels       map[string]string `yaml:"labels" json:"labels"`
}

type CSIDriver struct {
	Name       string                 `yaml:"name" json:"name"`
	Repository string                 `yaml:"repository" json:"repository"`
	Version    string                 `yaml:"version" json:"version"`
	Config     map[string]interface{} `yaml:"config" json:"config"`
}

type APIServerConfig struct {
	ExtraArgs        map[string]string `yaml:"extraArgs" json:"extraArgs"`
	ExtraVolumes     []VolumeMount     `yaml:"extraVolumes" json:"extraVolumes"`
	AuditLog         bool              `yaml:"auditLog" json:"auditLog"`
	EncryptionConfig bool              `yaml:"encryptionConfig" json:"encryptionConfig"`
}

type ControllerConfig struct {
	ExtraArgs    map[string]string `yaml:"extraArgs" json:"extraArgs"`
	ExtraVolumes []VolumeMount     `yaml:"extraVolumes" json:"extraVolumes"`
}

type SchedulerConfig struct {
	ExtraArgs    map[string]string `yaml:"extraArgs" json:"extraArgs"`
	ExtraVolumes []VolumeMount     `yaml:"extraVolumes" json:"extraVolumes"`
	Profile      string            `yaml:"profile" json:"profile"`
}

type KubeletConfig struct {
	ExtraArgs       map[string]string `yaml:"extraArgs" json:"extraArgs"`
	ExtraVolumes    []VolumeMount     `yaml:"extraVolumes" json:"extraVolumes"`
	ClusterDNS      string            `yaml:"clusterDns" json:"clusterDns"`
	ClusterDomain   string            `yaml:"clusterDomain" json:"clusterDomain"`
	RegistryMirrors []string          `yaml:"registryMirrors" json:"registryMirrors"`
}

type EtcdConfig struct {
	External        bool            `yaml:"external" json:"external"`
	Endpoints       []string        `yaml:"endpoints" json:"endpoints"`
	CAFile          string          `yaml:"caFile" json:"caFile"`
	CertFile        string          `yaml:"certFile" json:"certFile"`
	KeyFile         string          `yaml:"keyFile" json:"keyFile"`
	BackupInterval  time.Duration   `yaml:"backupInterval" json:"backupInterval"`
	BackupRetention int             `yaml:"backupRetention" json:"backupRetention"`
	Snapshot        *SnapshotConfig `yaml:"snapshot,omitempty" json:"snapshot,omitempty"`
}

type SnapshotConfig struct {
	Schedule  string `yaml:"schedule" json:"schedule"`
	Retention int    `yaml:"retention" json:"retention"`
	S3Bucket  string `yaml:"s3Bucket" json:"s3Bucket"`
}

type VolumeMount struct {
	Name      string `yaml:"name" json:"name"`
	MountPath string `yaml:"mountPath" json:"mountPath"`
	HostPath  string `yaml:"hostPath" json:"hostPath"`
	ReadOnly  bool   `yaml:"readOnly" json:"readOnly"`
}

type AddonConfig struct {
	Name       string                 `yaml:"name" json:"name"`
	Enabled    bool                   `yaml:"enabled" json:"enabled"`
	Version    string                 `yaml:"version" json:"version"`
	Namespace  string                 `yaml:"namespace" json:"namespace"`
	Values     map[string]interface{} `yaml:"values" json:"values"`
	Repository string                 `yaml:"repository" json:"repository"`
}

type AdmissionConfig struct {
	Plugins []string          `yaml:"plugins" json:"plugins"`
	Config  map[string]string `yaml:"config" json:"config"`
}

// Provider-specific VPC configurations

// DOVPCConfig - DigitalOcean specific VPC settings
type DOVPCConfig struct {
	IPRange     string `yaml:"ipRange" json:"ipRange"`         // IP range for VPC
	Description string `yaml:"description" json:"description"` // VPC description
}

// LinodeVPCConfig - Linode specific VPC settings
type LinodeVPCConfig struct {
	Label       string               `yaml:"label" json:"label"`             // VPC label
	Description string               `yaml:"description" json:"description"` // VPC description
	Subnets     []LinodeSubnetConfig `yaml:"subnets" json:"subnets"`         // VPC subnets
}

// LinodeSubnetConfig - Linode VPC subnet configuration
type LinodeSubnetConfig struct {
	Label string `yaml:"label" json:"label"` // Subnet label
	IPv4  string `yaml:"ipv4" json:"ipv4"`   // IPv4 CIDR
}
