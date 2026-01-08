; Sloth Kubernetes - AWS Cluster Configuration with Tailscale VPN
; This is a complete example of an AWS cluster with Tailscale mesh VPN via Headscale
; Auto-provisions a Headscale coordination server

(cluster
  ; Cluster metadata
  (metadata
    (name (concat "sloth-tailscale-" (env "CLUSTER_SUFFIX" "cluster")))
    (environment (env "CLUSTER_ENV" "production"))
    (description "Kubernetes cluster on AWS with Tailscale/Headscale mesh VPN")
    (owner (env "CLUSTER_OWNER" "chalkan3"))
    (labels
      (project "sloth-kubernetes")
      (provider "aws")
      (vpn "tailscale")
      (created-at (now))))

  ; Cluster specification
  (cluster
    (type "rke2")
    (version (env "RKE2_VERSION" "v1.29.0+rke2r1"))
    (high-availability true))

  ; Cloud providers
  (providers
    (aws
      (enabled true)
      (region (env "AWS_REGION" "us-east-1"))))

  ; Network configuration - Using Tailscale instead of WireGuard
  (network
    (mode "tailscale")
    (pod-cidr (env "POD_CIDR" "10.42.0.0/16"))
    (service-cidr (env "SERVICE_CIDR" "10.43.0.0/16"))

    ; Tailscale configuration with auto-provisioned Headscale
    (tailscale
      (enabled true)
      (create true)                                    ; Auto-provision Headscale server
      (provider "aws")                                 ; Provider for Headscale server
      (region (env "AWS_REGION" "us-east-1"))         ; Region for Headscale server
      (size (env "HEADSCALE_SIZE" "t3.small"))        ; Instance type for Headscale
      (namespace "kubernetes")                         ; Tailnet namespace
      (accept-routes true)                             ; Accept advertised routes
      (tags ("tag:k8s-node" "tag:production"))))      ; ACL tags for nodes

  ; Security configuration
  (security
    (ssh
      (key-path (env "SSH_KEY_PATH" "~/.ssh/id_rsa"))
      (public-key-path (env "SSH_PUBLIC_KEY_PATH" "~/.ssh/id_rsa.pub"))
      (port 22))
    (bastion
      (enabled true)
      (provider "aws")
      (region (env "AWS_REGION" "us-east-1"))
      (size "t3.micro")
      (name (concat "sloth-bastion-" (uuid-short)))
      (ssh-port 22)
      (allowed-cidrs (env "BASTION_ALLOWED_CIDRS" "0.0.0.0/0"))
      (enable-audit-log true)
      (idle-timeout 30)
      (max-sessions 10)))

  ; Node pools
  (node-pools
    (masters
      (name "aws-masters")
      (provider "aws")
      (region (env "AWS_REGION" "us-east-1"))
      (count (default (env "MASTER_COUNT") 3))
      (roles master etcd)
      (size (env "AWS_MASTER_SIZE" "t3.medium"))
      (labels
        (role "control-plane")
        (tier "master")))
    (workers
      (name "aws-workers")
      (provider "aws")
      (region (env "AWS_REGION" "us-east-1"))
      (count (default (env "WORKER_COUNT") 3))
      (roles worker)
      (size (env "AWS_WORKER_SIZE" "t3.large"))
      (spot-instance (env "USE_SPOT_INSTANCES" false))
      (labels
        (role "worker")
        (tier "application"))))

  ; Kubernetes configuration
  (kubernetes
    (version (env "K8S_VERSION" "v1.29.0"))
    (distribution "rke2")
    (network-plugin (env "CNI_PLUGIN" "canal"))
    (pod-cidr (env "POD_CIDR" "10.42.0.0/16"))
    (service-cidr (env "SERVICE_CIDR" "10.43.0.0/16"))
    (cluster-domain "cluster.local")
    (rke2
      (version (env "RKE2_VERSION" "v1.29.0+rke2r1"))
      (channel "stable")
      (disable-components "rke2-ingress-nginx")
      (snapshot-schedule-cron (env "SNAPSHOT_CRON" "0 */6 * * *"))
      (snapshot-retention (default (env "SNAPSHOT_RETENTION") 5))))

  ; Monitoring
  (monitoring
    (enabled (env "MONITORING_ENABLED" true))
    (prometheus
      (enabled true)
      (retention (env "PROMETHEUS_RETENTION" "15d")))
    (grafana
      (enabled true)))

  ; Addons configuration
  (addons
    ; Salt Master/Minion for configuration management
    (salt
      (enabled (env "SALT_ENABLED" true))
      (master-node "0")        ; Install on first master (index 0)
      (api-enabled true)       ; Enable Salt API
      (api-port (default (env "SALT_API_PORT") 8000))
      (secure-auth true)       ; Use hash-based secure authentication
      (auto-join true)         ; All nodes auto-join as minions
      (audit-logging true))))
