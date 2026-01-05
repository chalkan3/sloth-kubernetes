package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/benchmark"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var (
	benchmarkType       string
	benchmarkOutput     string
	benchmarkVerbose    bool
	benchmarkKubeconfig string
	benchmarkNodes      []string
	benchmarkCompare    string
	benchmarkSaveFile   string
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run cluster performance benchmarks",
	Long: `Run comprehensive performance benchmarks on your Kubernetes cluster.

Available benchmark types:
  - network: Pod-to-pod latency, DNS resolution, throughput, CNI performance
  - storage: IOPS, sequential I/O, random I/O, PVC provisioning
  - cpu: CPU efficiency, scheduling latency, API server response
  - memory: Memory utilization, bandwidth, etcd usage
  - all: Run all benchmarks (default)

The benchmarks measure real-world performance characteristics and compare
them against Kubernetes best practices and reference values.`,
	Example: `  # Run all benchmarks
  sloth-kubernetes benchmark run

  # Run only network benchmarks
  sloth-kubernetes benchmark run --type network

  # Run storage benchmarks with verbose output
  sloth-kubernetes benchmark run --type storage --verbose

  # Save results to JSON file
  sloth-kubernetes benchmark run --output json --save results.json

  # Show quick benchmark summary
  sloth-kubernetes benchmark quick

  # Compare with previous results
  sloth-kubernetes benchmark run --compare previous-results.json`,
}

var benchmarkRunCmd = &cobra.Command{
	Use:   "run [stack-name]",
	Short: "Execute benchmarks",
	Long:  `Execute the specified benchmarks and display results.`,
	RunE:  runBenchmark,
}

var benchmarkQuickCmd = &cobra.Command{
	Use:   "quick [stack-name]",
	Short: "Quick benchmark summary",
	Long:  `Run a quick set of essential benchmarks for a fast performance overview.`,
	RunE:  runQuickBenchmark,
}

var benchmarkReportCmd = &cobra.Command{
	Use:   "report [file]",
	Short: "Display benchmark report from file",
	Long:  `Load and display a benchmark report from a previously saved JSON file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBenchmarkReport,
}

var benchmarkCompareCmd = &cobra.Command{
	Use:   "compare <file1> <file2>",
	Short: "Compare two benchmark reports",
	Long:  `Compare two benchmark result files and show differences.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runBenchmarkCompare,
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.AddCommand(benchmarkRunCmd)
	benchmarkCmd.AddCommand(benchmarkQuickCmd)
	benchmarkCmd.AddCommand(benchmarkReportCmd)
	benchmarkCmd.AddCommand(benchmarkCompareCmd)

	// Run command flags
	benchmarkRunCmd.Flags().StringVar(&benchmarkType, "type", "all", "Benchmark type (network, storage, cpu, memory, all)")
	benchmarkRunCmd.Flags().StringVar(&benchmarkOutput, "output", "text", "Output format (text, json, compact)")
	benchmarkRunCmd.Flags().BoolVarP(&benchmarkVerbose, "verbose", "v", false, "Show verbose output with details")
	benchmarkRunCmd.Flags().StringVar(&benchmarkKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	benchmarkRunCmd.Flags().StringSliceVar(&benchmarkNodes, "nodes", []string{}, "Specific nodes to benchmark")
	benchmarkRunCmd.Flags().StringVar(&benchmarkSaveFile, "save", "", "Save results to file")
	benchmarkRunCmd.Flags().StringVar(&benchmarkCompare, "compare", "", "Compare with previous results file")

	// Quick command flags
	benchmarkQuickCmd.Flags().StringVar(&benchmarkKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	benchmarkQuickCmd.Flags().StringVar(&benchmarkOutput, "output", "compact", "Output format (text, json, compact)")

	// Compare command flags
	benchmarkCompareCmd.Flags().StringVar(&benchmarkOutput, "output", "text", "Output format (text, json)")
}

func createBenchmarkManager(targetStack string) (*benchmark.Manager, error) {
	// Get stack info including SSH credentials
	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack info: %w", err)
	}

	// Get kubeconfig from stack
	kubeconfigPath, err := GetKubeconfigFromStack(targetStack)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig from stack '%s': %w", targetStack, err)
	}

	manager := benchmark.NewManager(stackInfo.MasterIP, "", kubeconfigPath)
	manager.SetVerbose(benchmarkVerbose)

	if len(benchmarkNodes) > 0 {
		manager.SetNodeFilter(benchmarkNodes)
	}

	return manager, nil
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	printHeader("Cluster Benchmark")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBenchmarkManager(targetStack)
	if err != nil {
		return err
	}

	// Parse benchmark type
	var bType benchmark.BenchmarkType
	switch strings.ToLower(benchmarkType) {
	case "network":
		bType = benchmark.BenchmarkNetwork
	case "storage":
		bType = benchmark.BenchmarkStorage
	case "cpu":
		bType = benchmark.BenchmarkCPU
	case "memory":
		bType = benchmark.BenchmarkMemory
	case "all":
		bType = benchmark.BenchmarkAll
	default:
		return fmt.Errorf("invalid benchmark type: %s (valid: network, storage, cpu, memory, all)", benchmarkType)
	}

	fmt.Println()
	color.Cyan("Running %s benchmarks...", benchmarkType)
	fmt.Println()

	startTime := time.Now()
	report, err := manager.RunBenchmark(bType)
	if err != nil {
		operations.RecordBenchmark(targetStack, benchmarkType, 0, "", 0, 0, 0, 0, nil, time.Since(startTime), err)
		return fmt.Errorf("benchmark failed: %w", err)
	}

	if benchmarkVerbose {
		fmt.Printf("Benchmarks completed in %s\n", time.Since(startTime))
	}

	// Record benchmark to history
	operations.RecordBenchmark(targetStack, benchmarkType, report.Summary.OverallScore, report.Summary.Grade, report.Summary.NetworkScore, report.Summary.StorageScore, report.Summary.CPUScore, report.Summary.MemoryScore, report.Summary.Recommendations, time.Since(startTime), nil)

	// Output results
	switch benchmarkOutput {
	case "json":
		return outputBenchmarkJSON(report)
	case "compact":
		report.PrintCompact()
	default:
		report.PrintReport()
	}

	// Save to file if requested
	if benchmarkSaveFile != "" {
		if err := saveReport(report, benchmarkSaveFile); err != nil {
			color.Yellow("Warning: Failed to save results: %v", err)
		} else {
			fmt.Printf("\nResults saved to: %s\n", benchmarkSaveFile)
		}
	}

	// Compare with previous if requested
	if benchmarkCompare != "" {
		fmt.Println()
		if err := compareBenchmarks(report, benchmarkCompare); err != nil {
			color.Yellow("Warning: Failed to compare: %v", err)
		}
	}

	return nil
}

func runQuickBenchmark(cmd *cobra.Command, args []string) error {
	printHeader("Quick Benchmark")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBenchmarkManager(targetStack)
	if err != nil {
		return err
	}

	// Run all benchmarks but show compact output
	startTime := time.Now()
	report, err := manager.RunBenchmark(benchmark.BenchmarkAll)
	if err != nil {
		operations.RecordBenchmark(targetStack, "quick", 0, "", 0, 0, 0, 0, nil, time.Since(startTime), err)
		return fmt.Errorf("benchmark failed: %w", err)
	}

	// Record benchmark to history
	operations.RecordBenchmark(targetStack, "quick", report.Summary.OverallScore, report.Summary.Grade, report.Summary.NetworkScore, report.Summary.StorageScore, report.Summary.CPUScore, report.Summary.MemoryScore, report.Summary.Recommendations, time.Since(startTime), nil)

	if benchmarkOutput == "json" {
		return outputBenchmarkJSON(report)
	}

	// Print compact summary
	fmt.Println()
	report.PrintCompact()
	fmt.Println()

	// Print key metrics only
	color.Cyan("Key Metrics:")
	for _, result := range report.Results {
		switch result.Name {
		case "Pod-to-Pod Latency", "Storage IOPS", "API Server Latency":
			statusIcon := "[OK]"
			if result.Status == benchmark.StatusFailed {
				statusIcon = "[FAIL]"
			}
			fmt.Printf("  %s %-25s %8.2f %s\n",
				statusIcon,
				result.Name,
				result.Score,
				result.Unit)
		}
	}

	// Print top recommendation if any
	if len(report.Summary.Recommendations) > 0 && report.Summary.OverallScore < 80 {
		fmt.Println()
		color.Yellow("Top Recommendation: %s", report.Summary.Recommendations[0])
	}

	return nil
}

func runBenchmarkReport(cmd *cobra.Command, args []string) error {
	filename := args[0]

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var report benchmark.BenchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("failed to parse report: %w", err)
	}

	report.PrintReport()
	return nil
}

func runBenchmarkCompare(cmd *cobra.Command, args []string) error {
	file1, file2 := args[0], args[1]

	report1, err := loadReport(file1)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", file1, err)
	}

	report2, err := loadReport(file2)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", file2, err)
	}

	printComparison(report1, report2, file1, file2)
	return nil
}

func outputBenchmarkJSON(report *benchmark.BenchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func saveReport(report *benchmark.BenchmarkReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func loadReport(filename string) (*benchmark.BenchmarkReport, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var report benchmark.BenchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	return &report, nil
}

func compareBenchmarks(current *benchmark.BenchmarkReport, previousFile string) error {
	previous, err := loadReport(previousFile)
	if err != nil {
		return err
	}

	printComparison(previous, current, "Previous", "Current")
	return nil
}

func printComparison(report1, report2 *benchmark.BenchmarkReport, name1, name2 string) {
	color.Cyan("Benchmark Comparison")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-30s vs %-30s\n", name1, name2)
	fmt.Println()

	// Compare overall scores
	scoreDiff := report2.Summary.OverallScore - report1.Summary.OverallScore
	var diffColor *color.Color
	var diffSign string
	if scoreDiff > 0 {
		diffColor = color.New(color.FgGreen)
		diffSign = "+"
	} else if scoreDiff < 0 {
		diffColor = color.New(color.FgRed)
		diffSign = ""
	} else {
		diffColor = color.New(color.FgWhite)
		diffSign = ""
	}

	fmt.Printf("Overall Score: %.1f -> %.1f (", report1.Summary.OverallScore, report2.Summary.OverallScore)
	diffColor.Printf("%s%.1f", diffSign, scoreDiff)
	fmt.Println(")")

	fmt.Printf("Grade: %s -> %s\n", report1.Summary.Grade, report2.Summary.Grade)

	fmt.Println()
	color.Cyan("Category Comparison")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("%-15s %10s %10s %10s\n", "Category", name1, name2, "Change")
	fmt.Println(strings.Repeat("-", 50))

	printCategoryComparison("Network", report1.Summary.NetworkScore, report2.Summary.NetworkScore)
	printCategoryComparison("Storage", report1.Summary.StorageScore, report2.Summary.StorageScore)
	printCategoryComparison("CPU", report1.Summary.CPUScore, report2.Summary.CPUScore)
	printCategoryComparison("Memory", report1.Summary.MemoryScore, report2.Summary.MemoryScore)

	// Compare individual metrics
	fmt.Println()
	color.Cyan("Metric Changes")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("%-30s %15s %15s %10s\n", "Metric", name1, name2, "Change")
	fmt.Println(strings.Repeat("-", 70))

	// Build maps for comparison
	metrics1 := make(map[string]benchmark.BenchmarkResult)
	metrics2 := make(map[string]benchmark.BenchmarkResult)

	for _, r := range report1.Results {
		metrics1[r.Name] = r
	}
	for _, r := range report2.Results {
		metrics2[r.Name] = r
	}

	// Compare all metrics
	for name, result2 := range metrics2 {
		result1, exists := metrics1[name]
		if !exists {
			continue
		}

		diff := result2.Score - result1.Score
		percentChange := 0.0
		if result1.Score != 0 {
			percentChange = (diff / result1.Score) * 100
		}

		// Determine if change is improvement or regression
		isLowerBetter := strings.Contains(name, "Latency") || strings.Contains(name, "Time") || strings.Contains(name, "Usage")
		improved := (isLowerBetter && diff < 0) || (!isLowerBetter && diff > 0)

		var changeColor *color.Color
		if improved {
			changeColor = color.New(color.FgGreen)
		} else if diff != 0 {
			changeColor = color.New(color.FgRed)
		} else {
			changeColor = color.New(color.FgWhite)
		}

		changeStr := fmt.Sprintf("%+.1f%%", percentChange)
		fmt.Printf("%-30s %12.2f %s %12.2f %s ",
			name,
			result1.Score, result1.Unit,
			result2.Score, result2.Unit)
		changeColor.Printf("%10s\n", changeStr)
	}
}

func printCategoryComparison(category string, score1, score2 float64) {
	diff := score2 - score1
	var diffColor *color.Color
	if diff > 0 {
		diffColor = color.New(color.FgGreen)
	} else if diff < 0 {
		diffColor = color.New(color.FgRed)
	} else {
		diffColor = color.New(color.FgWhite)
	}

	fmt.Printf("%-15s %10.1f %10.1f ", category, score1, score2)
	diffColor.Printf("%+10.1f\n", diff)
}
