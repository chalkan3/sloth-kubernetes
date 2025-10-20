package addons

// GetAddonCatalog returns all available addons
func GetAddonCatalog() map[string]*Addon {
	return map[string]*Addon{
		"ingress-nginx": {
			Name:        "ingress-nginx",
			DisplayName: "NGINX Ingress Controller",
			Description: "Production-grade Ingress controller for Kubernetes",
			Category:    string(CategoryIngress),
			Version:     "4.8.3",
			Chart:       "ingress-nginx",
			Repository:  "https://kubernetes.github.io/ingress-nginx",
			Namespace:   "ingress-nginx",
			Website:     "https://kubernetes.github.io/ingress-nginx/",
			Docs:        "https://kubernetes.github.io/ingress-nginx/deploy/",
		},
		"cert-manager": {
			Name:        "cert-manager",
			DisplayName: "Cert Manager",
			Description: "Automatic TLS certificate management for Kubernetes",
			Category:    string(CategorySecurity),
			Version:     "v1.13.3",
			Chart:       "cert-manager",
			Repository:  "https://charts.jetstack.io",
			Namespace:   "cert-manager",
			Website:     "https://cert-manager.io/",
			Docs:        "https://cert-manager.io/docs/",
		},
		"prometheus": {
			Name:        "prometheus",
			DisplayName: "Prometheus Stack",
			Description: "Complete monitoring stack with Prometheus, Grafana, and Alertmanager",
			Category:    string(CategoryMonitoring),
			Version:     "55.5.0",
			Chart:       "kube-prometheus-stack",
			Repository:  "https://prometheus-community.github.io/helm-charts",
			Namespace:   "monitoring",
			Website:     "https://prometheus.io/",
			Docs:        "https://github.com/prometheus-operator/kube-prometheus",
		},
		"longhorn": {
			Name:        "longhorn",
			DisplayName: "Longhorn Storage",
			Description: "Cloud native distributed block storage for Kubernetes",
			Category:    string(CategoryStorage),
			Version:     "1.5.3",
			Chart:       "longhorn",
			Repository:  "https://charts.longhorn.io",
			Namespace:   "longhorn-system",
			Website:     "https://longhorn.io/",
			Docs:        "https://longhorn.io/docs/",
		},
		"argocd": {
			Name:        "argocd",
			DisplayName: "Argo CD",
			Description: "Declarative GitOps continuous delivery tool for Kubernetes",
			Category:    string(CategoryCD),
			Version:     "5.51.6",
			Chart:       "argo-cd",
			Repository:  "https://argoproj.github.io/argo-helm",
			Namespace:   "argocd",
			Website:     "https://argo-cd.readthedocs.io/",
			Docs:        "https://argo-cd.readthedocs.io/en/stable/",
		},
		"loki": {
			Name:        "loki",
			DisplayName: "Loki Stack",
			Description: "Log aggregation system designed to work with Grafana",
			Category:    string(CategoryLogging),
			Version:     "2.9.11",
			Chart:       "loki-stack",
			Repository:  "https://grafana.github.io/helm-charts",
			Namespace:   "logging",
			Website:     "https://grafana.com/oss/loki/",
			Docs:        "https://grafana.com/docs/loki/latest/",
		},
		"metallb": {
			Name:        "metallb",
			DisplayName: "MetalLB",
			Description: "Load-balancer implementation for bare metal Kubernetes clusters",
			Category:    string(CategoryNetworking),
			Version:     "0.13.12",
			Chart:       "metallb",
			Repository:  "https://metallb.github.io/metallb",
			Namespace:   "metallb-system",
			Website:     "https://metallb.universe.tf/",
			Docs:        "https://metallb.universe.tf/installation/",
		},
		"postgres-operator": {
			Name:        "postgres-operator",
			DisplayName: "PostgreSQL Operator",
			Description: "Create and manage PostgreSQL clusters in Kubernetes",
			Category:    string(CategoryDatabase),
			Version:     "1.11.0",
			Chart:       "postgres-operator",
			Repository:  "https://opensource.zalando.com/postgres-operator/charts/postgres-operator",
			Namespace:   "postgres-operator",
			Website:     "https://postgres-operator.readthedocs.io/",
			Docs:        "https://postgres-operator.readthedocs.io/en/latest/",
		},
		"istio": {
			Name:        "istio",
			DisplayName: "Istio Service Mesh",
			Description: "Service mesh for microservices with load balancing, service-to-service authentication, and monitoring",
			Category:    string(CategoryNetworking),
			Version:     "1.20.1",
			Chart:       "base",
			Repository:  "https://istio-release.storage.googleapis.com/charts",
			Namespace:   "istio-system",
			Website:     "https://istio.io/",
			Docs:        "https://istio.io/latest/docs/",
		},
		"external-dns": {
			Name:        "external-dns",
			DisplayName: "External DNS",
			Description: "Automatically manage DNS records from Kubernetes resources",
			Category:    string(CategoryNetworking),
			Version:     "1.14.0",
			Chart:       "external-dns",
			Repository:  "https://kubernetes-sigs.github.io/external-dns/",
			Namespace:   "external-dns",
			Website:     "https://github.com/kubernetes-sigs/external-dns",
			Docs:        "https://kubernetes-sigs.github.io/external-dns/",
		},
		"velero": {
			Name:        "velero",
			DisplayName: "Velero Backup",
			Description: "Backup and migrate Kubernetes cluster resources and persistent volumes",
			Category:    string(CategoryStorage),
			Version:     "5.2.0",
			Chart:       "velero",
			Repository:  "https://vmware-tanzu.github.io/helm-charts",
			Namespace:   "velero",
			Website:     "https://velero.io/",
			Docs:        "https://velero.io/docs/",
		},
		"sealed-secrets": {
			Name:        "sealed-secrets",
			DisplayName: "Sealed Secrets",
			Description: "Encrypt your Secret into a SealedSecret for safe storage in Git",
			Category:    string(CategorySecurity),
			Version:     "2.14.1",
			Chart:       "sealed-secrets",
			Repository:  "https://bitnami-labs.github.io/sealed-secrets",
			Namespace:   "kube-system",
			Website:     "https://sealed-secrets.netlify.app/",
			Docs:        "https://github.com/bitnami-labs/sealed-secrets",
		},
	}
}

// GetAddon returns a specific addon by name
func GetAddon(name string) (*Addon, bool) {
	catalog := GetAddonCatalog()
	addon, exists := catalog[name]
	return addon, exists
}

// GetAddonsByCategory returns all addons in a specific category
func GetAddonsByCategory(category Category) []*Addon {
	catalog := GetAddonCatalog()
	var addons []*Addon

	for _, addon := range catalog {
		if addon.Category == string(category) {
			addons = append(addons, addon)
		}
	}

	return addons
}

// GetCategories returns all available categories
func GetCategories() []string {
	return []string{
		string(CategoryIngress),
		string(CategoryStorage),
		string(CategoryMonitoring),
		string(CategorySecurity),
		string(CategoryNetworking),
		string(CategoryCD),
		string(CategoryLogging),
		string(CategoryDatabase),
	}
}
