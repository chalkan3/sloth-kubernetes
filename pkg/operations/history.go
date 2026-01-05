// Package operations provides functionality for tracking and managing
// CLI operation history in Pulumi state.
package operations

import (
	"sync"
	"time"
)

const (
	// DefaultMaxEntries is the default maximum number of entries per operation type
	DefaultMaxEntries = 50
)

// OperationsHistory holds the history of all CLI operations
type OperationsHistory struct {
	BackupHistory    []BackupEntry    `json:"backupHistory"`
	UpgradeHistory   []UpgradeEntry   `json:"upgradeHistory"`
	HealthHistory    []HealthEntry    `json:"healthHistory"`
	BenchmarkHistory []BenchmarkEntry `json:"benchmarkHistory"`
	MaxEntries       int              `json:"maxEntries"`
	LastUpdated      time.Time        `json:"lastUpdated"`
	mu               sync.Mutex       `json:"-"`
}

// BackupEntry represents a single backup operation record
type BackupEntry struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Operation       string    `json:"operation"` // create, restore, delete, schedule-create, schedule-delete
	BackupName      string    `json:"backupName"`
	Status          string    `json:"status"` // success, failed, in-progress
	Namespaces      []string  `json:"namespaces,omitempty"`
	IncludedItems   int       `json:"includedItems,omitempty"`
	ExcludedItems   int       `json:"excludedItems,omitempty"`
	Duration        string    `json:"duration,omitempty"`
	StorageLocation string    `json:"storageLocation,omitempty"`
	TTL             string    `json:"ttl,omitempty"`
	Error           string    `json:"error,omitempty"`
}

// UpgradeEntry represents a single upgrade operation record
type UpgradeEntry struct {
	ID          string              `json:"id"`
	Timestamp   time.Time           `json:"timestamp"`
	Operation   string              `json:"operation"` // upgrade, rollback
	FromVersion string              `json:"fromVersion"`
	ToVersion   string              `json:"toVersion"`
	Strategy    string              `json:"strategy,omitempty"` // rolling, blue-green, canary, in-place
	Status      string              `json:"status"`             // success, failed, partial
	NodesTotal  int                 `json:"nodesTotal"`
	NodesOK     int                 `json:"nodesOk"`
	NodesFailed int                 `json:"nodesFailed"`
	Duration    string              `json:"duration,omitempty"`
	NodeResults []UpgradeNodeResult `json:"nodeResults,omitempty"`
	Error       string              `json:"error,omitempty"`
}

// UpgradeNodeResult represents the result of upgrading a single node
type UpgradeNodeResult struct {
	NodeName    string `json:"nodeName"`
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	Status      string `json:"status"` // success, failed, skipped
	Duration    string `json:"duration,omitempty"`
	Error       string `json:"error,omitempty"`
}

// HealthEntry represents a single health check record
type HealthEntry struct {
	ID            string             `json:"id"`
	Timestamp     time.Time          `json:"timestamp"`
	OverallStatus string             `json:"overallStatus"` // healthy, degraded, unhealthy
	ChecksRun     int                `json:"checksRun"`
	ChecksPassed  int                `json:"checksPassed"`
	ChecksWarning int                `json:"checksWarning"`
	ChecksFailed  int                `json:"checksFailed"`
	Duration      string             `json:"duration,omitempty"`
	Summary       string             `json:"summary,omitempty"`
	CheckResults  []HealthCheckEntry `json:"checkResults,omitempty"`
	Error         string             `json:"error,omitempty"`
}

// HealthCheckEntry represents a single health check within a health report
type HealthCheckEntry struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // passed, warning, failed
	Message  string `json:"message,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// BenchmarkEntry represents a single benchmark run record
type BenchmarkEntry struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	BenchmarkType   string                 `json:"benchmarkType"` // network, storage, cpu, memory, all, quick
	OverallScore    float64                `json:"overallScore"`
	Grade           string                 `json:"grade"` // A, B+, B, C+, C, D, F
	NetworkScore    float64                `json:"networkScore,omitempty"`
	StorageScore    float64                `json:"storageScore,omitempty"`
	CPUScore        float64                `json:"cpuScore,omitempty"`
	MemoryScore     float64                `json:"memoryScore,omitempty"`
	Duration        string                 `json:"duration,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
	Metrics         []BenchmarkMetricEntry `json:"metrics,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// BenchmarkMetricEntry represents a single metric within a benchmark
type BenchmarkMetricEntry struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Status     string  `json:"status"` // passed, warning, failed
	Reference  float64 `json:"reference,omitempty"`
	Percentage float64 `json:"percentage,omitempty"`
}

// NewOperationsHistory creates a new OperationsHistory with default settings
func NewOperationsHistory() *OperationsHistory {
	return &OperationsHistory{
		BackupHistory:    make([]BackupEntry, 0),
		UpgradeHistory:   make([]UpgradeEntry, 0),
		HealthHistory:    make([]HealthEntry, 0),
		BenchmarkHistory: make([]BenchmarkEntry, 0),
		MaxEntries:       DefaultMaxEntries,
		LastUpdated:      time.Now().UTC(),
	}
}

// AddBackup adds a backup entry to the history with FIFO pruning
func (h *OperationsHistory) AddBackup(entry BackupEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.BackupHistory = append(h.BackupHistory, entry)
	if len(h.BackupHistory) > h.MaxEntries {
		h.BackupHistory = h.BackupHistory[1:]
	}
	h.LastUpdated = time.Now().UTC()
}

// AddUpgrade adds an upgrade entry to the history with FIFO pruning
func (h *OperationsHistory) AddUpgrade(entry UpgradeEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.UpgradeHistory = append(h.UpgradeHistory, entry)
	if len(h.UpgradeHistory) > h.MaxEntries {
		h.UpgradeHistory = h.UpgradeHistory[1:]
	}
	h.LastUpdated = time.Now().UTC()
}

// AddHealth adds a health entry to the history with FIFO pruning
func (h *OperationsHistory) AddHealth(entry HealthEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.HealthHistory = append(h.HealthHistory, entry)
	if len(h.HealthHistory) > h.MaxEntries {
		h.HealthHistory = h.HealthHistory[1:]
	}
	h.LastUpdated = time.Now().UTC()
}

// AddBenchmark adds a benchmark entry to the history with FIFO pruning
func (h *OperationsHistory) AddBenchmark(entry BenchmarkEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.BenchmarkHistory = append(h.BenchmarkHistory, entry)
	if len(h.BenchmarkHistory) > h.MaxEntries {
		h.BenchmarkHistory = h.BenchmarkHistory[1:]
	}
	h.LastUpdated = time.Now().UTC()
}

// GetLatestBackup returns the most recent backup entry or nil
func (h *OperationsHistory) GetLatestBackup() *BackupEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.BackupHistory) == 0 {
		return nil
	}
	return &h.BackupHistory[len(h.BackupHistory)-1]
}

// GetLatestUpgrade returns the most recent upgrade entry or nil
func (h *OperationsHistory) GetLatestUpgrade() *UpgradeEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.UpgradeHistory) == 0 {
		return nil
	}
	return &h.UpgradeHistory[len(h.UpgradeHistory)-1]
}

// GetLatestHealth returns the most recent health entry or nil
func (h *OperationsHistory) GetLatestHealth() *HealthEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.HealthHistory) == 0 {
		return nil
	}
	return &h.HealthHistory[len(h.HealthHistory)-1]
}

// GetLatestBenchmark returns the most recent benchmark entry or nil
func (h *OperationsHistory) GetLatestBenchmark() *BenchmarkEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.BenchmarkHistory) == 0 {
		return nil
	}
	return &h.BenchmarkHistory[len(h.BenchmarkHistory)-1]
}

// GetBackupsByStatus returns all backup entries with the given status
func (h *OperationsHistory) GetBackupsByStatus(status string) []BackupEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []BackupEntry
	for _, entry := range h.BackupHistory {
		if entry.Status == status {
			result = append(result, entry)
		}
	}
	return result
}

// GetBackupsAfter returns all backup entries after the given time
func (h *OperationsHistory) GetBackupsAfter(t time.Time) []BackupEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []BackupEntry
	for _, entry := range h.BackupHistory {
		if entry.Timestamp.After(t) {
			result = append(result, entry)
		}
	}
	return result
}

// TotalOperations returns the total number of operations recorded
func (h *OperationsHistory) TotalOperations() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.BackupHistory) + len(h.UpgradeHistory) + len(h.HealthHistory) + len(h.BenchmarkHistory)
}

// Clear removes all entries from the history
func (h *OperationsHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.BackupHistory = make([]BackupEntry, 0)
	h.UpgradeHistory = make([]UpgradeEntry, 0)
	h.HealthHistory = make([]HealthEntry, 0)
	h.BenchmarkHistory = make([]BenchmarkEntry, 0)
	h.LastUpdated = time.Now().UTC()
}
