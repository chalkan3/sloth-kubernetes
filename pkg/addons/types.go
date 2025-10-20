package addons

// Addon represents a Kubernetes addon that can be installed
type Addon struct {
	Name         string
	DisplayName  string
	Description  string
	Category     string
	Version      string
	Chart        string // Helm chart name
	Repository   string // Helm repository URL
	Namespace    string
	Dependencies []string
	Values       map[string]interface{}
	InstallCmd   string // Alternative to Helm (kubectl apply, etc)
	Website      string
	Docs         string
}

// AddonStatus represents the installation status of an addon
type AddonStatus struct {
	Name      string
	Installed bool
	Version   string
	Namespace string
	Status    string // Running, Pending, Failed, Unknown
	Pods      int
	Ready     int
}

// Category represents addon categories
type Category string

const (
	CategoryIngress    Category = "ingress"
	CategoryStorage    Category = "storage"
	CategoryMonitoring Category = "monitoring"
	CategorySecurity   Category = "security"
	CategoryNetworking Category = "networking"
	CategoryCD         Category = "cd"
	CategoryLogging    Category = "logging"
	CategoryDatabase   Category = "database"
)
