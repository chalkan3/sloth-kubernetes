package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
)

// Global mutex to prevent concurrent state updates
var stateMutex sync.Mutex

// GetOperationsHistory retrieves the operations history from a Pulumi stack
func GetOperationsHistory(stackName string) (*OperationsHistory, error) {
	if stackName == "" {
		return nil, fmt.Errorf("stack name is required")
	}

	ctx := context.Background()

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Get outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Get operationsHistory from outputs
	historyOutput, ok := outputs["operationsHistory"]
	if !ok {
		// Return new empty history if not found
		return NewOperationsHistory(), nil
	}

	// Parse the history JSON
	historyStr, ok := historyOutput.Value.(string)
	if !ok {
		// Try to marshal if it's a map
		historyJSON, err := json.Marshal(historyOutput.Value)
		if err != nil {
			return NewOperationsHistory(), nil
		}
		historyStr = string(historyJSON)
	}

	if historyStr == "" || historyStr == "{}" {
		return NewOperationsHistory(), nil
	}

	var history OperationsHistory
	if err := json.Unmarshal([]byte(historyStr), &history); err != nil {
		// If parsing fails, return new history
		return NewOperationsHistory(), nil
	}

	return &history, nil
}

// SaveOperationsHistory saves the operations history to a Pulumi stack
func SaveOperationsHistory(stackName string, history *OperationsHistory) error {
	if stackName == "" {
		return fmt.Errorf("stack name is required")
	}

	stateMutex.Lock()
	defer stateMutex.Unlock()

	ctx := context.Background()

	// Create workspace with S3 support
	ws, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, ws)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export current state
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse deployment JSON
	var deploymentData map[string]interface{}
	if err := json.Unmarshal(deployment.Deployment, &deploymentData); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	// Find and update the stack resource outputs
	resources, ok := deploymentData["resources"].([]interface{})
	if !ok {
		return fmt.Errorf("resources not found in deployment")
	}

	// Marshal history to JSON
	history.LastUpdated = time.Now().UTC()
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Find the Stack resource and update its outputs
	found := false
	for i, res := range resources {
		resource, ok := res.(map[string]interface{})
		if !ok {
			continue
		}

		resType, _ := resource["type"].(string)
		if resType == "pulumi:pulumi:Stack" {
			// Update or create outputs
			outputs, ok := resource["outputs"].(map[string]interface{})
			if !ok {
				outputs = make(map[string]interface{})
			}
			outputs["operationsHistory"] = string(historyJSON)
			resource["outputs"] = outputs
			resources[i] = resource
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("stack resource not found in deployment")
	}

	// Marshal back to JSON
	deploymentData["resources"] = resources
	modifiedDeployment, err := json.Marshal(deploymentData)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	deployment.Deployment = modifiedDeployment

	// Import modified state back
	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import modified state: %w", err)
	}

	return nil
}

// AddBackupEntry adds a backup entry to the stack's operations history
func AddBackupEntry(stackName string, entry BackupEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		// Create new history if retrieval fails
		history = NewOperationsHistory()
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddBackup(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddUpgradeEntry adds an upgrade entry to the stack's operations history
func AddUpgradeEntry(stackName string, entry UpgradeEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddUpgrade(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddHealthEntry adds a health check entry to the stack's operations history
func AddHealthEntry(stackName string, entry HealthEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddHealth(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddBenchmarkEntry adds a benchmark entry to the stack's operations history
func AddBenchmarkEntry(stackName string, entry BenchmarkEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddBenchmark(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddNodeEntry adds a node entry to the stack's operations history
func AddNodeEntry(stackName string, entry NodeEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddNode(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddVPNEntry adds a VPN entry to the stack's operations history
func AddVPNEntry(stackName string, entry VPNEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddVPN(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddArgoCDEntry adds an ArgoCD entry to the stack's operations history
func AddArgoCDEntry(stackName string, entry ArgoCDEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddArgoCD(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddAddonsEntry adds an addons entry to the stack's operations history
func AddAddonsEntry(stackName string, entry AddonsEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddAddons(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddSaltEntry adds a Salt entry to the stack's operations history
func AddSaltEntry(stackName string, entry SaltEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddSalt(entry)

	return SaveOperationsHistory(stackName, history)
}

// AddValidationEntry adds a validation entry to the stack's operations history
func AddValidationEntry(stackName string, entry ValidationEntry) error {
	history, err := GetOperationsHistory(stackName)
	if err != nil {
		history = NewOperationsHistory()
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	history.AddValidation(entry)

	return SaveOperationsHistory(stackName, history)
}

// createWorkspaceWithS3Support creates a Pulumi workspace with S3/MinIO backend support
func createWorkspaceWithS3Support(ctx context.Context) (auto.Workspace, error) {
	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	projectName := "sloth-kubernetes"

	// Build project configuration with optional backend
	project := workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	// If PULUMI_BACKEND_URL is set, configure backend in the project
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		project.Backend = &workspace.ProjectBackend{
			URL: backendURL,
		}
	}

	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Project(project),
	}

	// Collect all AWS/S3 environment variables to pass to Pulumi subprocess
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"AWS_S3_ENDPOINT",
		"AWS_S3_USE_PATH_STYLE",
		"AWS_S3_FORCE_PATH_STYLE",
		"PULUMI_BACKEND_URL",
		"PULUMI_CONFIG_PASSPHRASE",
	}
	for _, key := range awsEnvKeys {
		if val := os.Getenv(key); val != "" {
			envVars[key] = val
		}
	}

	// Add environment variables to workspace options
	if len(envVars) > 0 {
		workspaceOpts = append(workspaceOpts, auto.EnvVars(envVars))
	}

	// If PULUMI_BACKEND_URL is set, use passphrase secrets provider
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		workspaceOpts = append(workspaceOpts, auto.SecretsProvider("passphrase"))
		if os.Getenv("PULUMI_CONFIG_PASSPHRASE") == "" {
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
			envVars["PULUMI_CONFIG_PASSPHRASE"] = ""
		}
	}

	return auto.NewLocalWorkspace(ctx, workspaceOpts...)
}

// RecordBackupOperation is a convenience function to record a backup operation
func RecordBackupOperation(stackName, operation, backupName, status string, namespaces []string, duration time.Duration, err error) {
	entry := BackupEntry{
		ID:         uuid.New().String(),
		Timestamp:  time.Now().UTC(),
		Operation:  operation,
		BackupName: backupName,
		Status:     status,
		Namespaces: namespaces,
		Duration:   duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddBackupEntry(stackName, entry); saveErr != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordUpgradeOperation is a convenience function to record an upgrade operation
func RecordUpgradeOperation(stackName, operation, fromVersion, toVersion, strategy, status string, nodesTotal, nodesOK, nodesFailed int, duration time.Duration, err error) {
	entry := UpgradeEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UTC(),
		Operation:   operation,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Strategy:    strategy,
		Status:      status,
		NodesTotal:  nodesTotal,
		NodesOK:     nodesOK,
		NodesFailed: nodesFailed,
		Duration:    duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddUpgradeEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordHealthCheck is a convenience function to record a health check
func RecordHealthCheck(stackName, overallStatus string, checksRun, checksPassed, checksWarning, checksFailed int, summary string, duration time.Duration, err error) {
	entry := HealthEntry{
		ID:            uuid.New().String(),
		Timestamp:     time.Now().UTC(),
		OverallStatus: overallStatus,
		ChecksRun:     checksRun,
		ChecksPassed:  checksPassed,
		ChecksWarning: checksWarning,
		ChecksFailed:  checksFailed,
		Summary:       summary,
		Duration:      duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddHealthEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordBenchmark is a convenience function to record a benchmark run
func RecordBenchmark(stackName, benchmarkType string, overallScore float64, grade string, networkScore, storageScore, cpuScore, memoryScore float64, recommendations []string, duration time.Duration, err error) {
	entry := BenchmarkEntry{
		ID:              uuid.New().String(),
		Timestamp:       time.Now().UTC(),
		BenchmarkType:   benchmarkType,
		OverallScore:    overallScore,
		Grade:           grade,
		NetworkScore:    networkScore,
		StorageScore:    storageScore,
		CPUScore:        cpuScore,
		MemoryScore:     memoryScore,
		Recommendations: recommendations,
		Duration:        duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddBenchmarkEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordNodeOperation is a convenience function to record a node operation
func RecordNodeOperation(stackName, operation, nodeName, nodeRole, nodeIP, status, details string, duration time.Duration, err error) {
	entry := NodeEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Operation: operation,
		NodeName:  nodeName,
		NodeRole:  nodeRole,
		NodeIP:    nodeIP,
		Status:    status,
		Details:   details,
		Duration:  duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddNodeEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordVPNOperation is a convenience function to record a VPN operation
func RecordVPNOperation(stackName, operation, nodeName, networkID, status, details string, nodesCount int, duration time.Duration, err error) {
	entry := VPNEntry{
		ID:         uuid.New().String(),
		Timestamp:  time.Now().UTC(),
		Operation:  operation,
		NodeName:   nodeName,
		NetworkID:  networkID,
		Status:     status,
		NodesCount: nodesCount,
		Details:    details,
		Duration:   duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddVPNEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordArgoCDOperation is a convenience function to record an ArgoCD operation
func RecordArgoCDOperation(stackName, operation, appName, namespace, status, syncStatus, healthState, details string, duration time.Duration, err error) {
	entry := ArgoCDEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UTC(),
		Operation:   operation,
		AppName:     appName,
		Namespace:   namespace,
		Status:      status,
		SyncStatus:  syncStatus,
		HealthState: healthState,
		Details:     details,
		Duration:    duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddArgoCDEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordAddonsOperation is a convenience function to record an addons operation
func RecordAddonsOperation(stackName, operation, addonName, addonType, status, details string, addonsApplied, addonsFailed int, duration time.Duration, err error) {
	entry := AddonsEntry{
		ID:            uuid.New().String(),
		Timestamp:     time.Now().UTC(),
		Operation:     operation,
		AddonName:     addonName,
		AddonType:     addonType,
		Status:        status,
		AddonsApplied: addonsApplied,
		AddonsFailed:  addonsFailed,
		Details:       details,
		Duration:      duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddAddonsEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordSaltOperation is a convenience function to record a Salt operation
func RecordSaltOperation(stackName, operation, target, function, arguments, status, output string, nodesTargeted, nodesSuccess, nodesFailed int, duration time.Duration, err error) {
	entry := SaltEntry{
		ID:            uuid.New().String(),
		Timestamp:     time.Now().UTC(),
		Operation:     operation,
		Target:        target,
		Function:      function,
		Arguments:     arguments,
		Status:        status,
		NodesTargeted: nodesTargeted,
		NodesSuccess:  nodesSuccess,
		NodesFailed:   nodesFailed,
		Output:        output,
		Duration:      duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddSaltEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}

// RecordValidation is a convenience function to record a validation run
func RecordValidation(stackName, validationType, overallStatus string, totalChecks, passedChecks, failedChecks, warningChecks int, duration time.Duration, err error) {
	entry := ValidationEntry{
		ID:             uuid.New().String(),
		Timestamp:      time.Now().UTC(),
		ValidationType: validationType,
		OverallStatus:  overallStatus,
		TotalChecks:    totalChecks,
		PassedChecks:   passedChecks,
		FailedChecks:   failedChecks,
		WarningChecks:  warningChecks,
		Duration:       duration.String(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if saveErr := AddValidationEntry(stackName, entry); saveErr != nil {
		fmt.Printf("Warning: Failed to save operation to history: %v\n", saveErr)
	}
}
