package main

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/internal/orchestrator"
	"kubernetes-create/pkg/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load config
		loader := config.NewLoader("config/cluster-config.yaml")
		clusterConfig, err := loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx.Log.Info("Starting REAL Kubernetes cluster deployment: WireGuard + RKE2 + DNS", nil)

		// Create SIMPLE orchestrator with ONLY REAL implementations (no mocks)
		_, err = orchestrator.NewSimpleRealOrchestratorComponent(ctx, "kubernetes-cluster", clusterConfig)
		if err != nil {
			return fmt.Errorf("failed to create orchestrator: %w", err)
		}

		return nil
	})
}
