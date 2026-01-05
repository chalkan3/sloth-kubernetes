---
title: Cluster Benchmarks
description: Run performance benchmarks on your Kubernetes cluster
sidebar_position: 7
---

# Cluster Benchmarks

sloth-kubernetes includes comprehensive benchmarking tools to measure and analyze your cluster's performance across network, storage, CPU, and memory dimensions.

## Overview

The benchmark system provides:
- **Multiple benchmark types**: Network, storage, CPU, memory, or all combined
- **Reference comparisons**: Results compared against Kubernetes best practices
- **Report saving**: Save results to JSON for historical tracking
- **Report comparison**: Compare benchmark results over time
- **Quick mode**: Fast performance overview

## Commands

### benchmark run

Execute comprehensive benchmarks on your cluster.

```bash
# Run all benchmarks
sloth-kubernetes benchmark run

# Run specific benchmark type
sloth-kubernetes benchmark run --type network

# Run with verbose output
sloth-kubernetes benchmark run --type storage --verbose

# Save results to JSON file
sloth-kubernetes benchmark run --output json --save results.json

# Compare with previous results
sloth-kubernetes benchmark run --compare previous-results.json
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--type` | Benchmark type (network, storage, cpu, memory, all) | `all` |
| `--output` | Output format (text, json, compact) | `text` |
| `--verbose`, `-v` | Show verbose output with details | `false` |
| `--kubeconfig` | Path to kubeconfig file | - |
| `--nodes` | Specific nodes to benchmark (comma-separated) | - |
| `--save` | Save results to file | - |
| `--compare` | Compare with previous results file | - |

### benchmark quick

Run a quick set of essential benchmarks for a fast performance overview.

```bash
sloth-kubernetes benchmark quick

# Output as JSON
sloth-kubernetes benchmark quick --output json
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--kubeconfig` | Path to kubeconfig file | - |
| `--output` | Output format (text, json, compact) | `compact` |

**Example output:**

```
═══════════════════════════════════════════════════════════════
                     QUICK BENCHMARK
═══════════════════════════════════════════════════════════════

Overall Score: 85.2/100  Grade: B+

Category Scores:
  Network:  88.0
  Storage:  82.5
  CPU:      86.3
  Memory:   84.0

Key Metrics:
  [OK] Pod-to-Pod Latency          0.45 ms
  [OK] Storage IOPS              15420 ops
  [OK] API Server Latency           12 ms

Top Recommendation: Consider increasing storage IOPS for better performance
```

### benchmark report

Display a benchmark report from a previously saved JSON file.

```bash
sloth-kubernetes benchmark report results.json
```

### benchmark compare

Compare two benchmark result files and show differences.

```bash
sloth-kubernetes benchmark compare baseline.json current.json
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output` | Output format (text, json) | `text` |

**Example output:**

```
Benchmark Comparison
======================================================================
baseline.json                  vs current.json

Overall Score: 78.5 -> 85.2 (+6.7)
Grade: B -> B+

Category Comparison
--------------------------------------------------
Category             baseline    current    Change
--------------------------------------------------
Network                 75.0       88.0     +13.0
Storage                 80.0       82.5      +2.5
CPU                     78.0       86.3      +8.3
Memory                  81.0       84.0      +3.0

Metric Changes
----------------------------------------------------------------------
Metric                             baseline        current    Change
----------------------------------------------------------------------
Pod-to-Pod Latency                0.65 ms         0.45 ms    -30.8%
Storage IOPS                    12500 ops       15420 ops    +23.4%
API Server Latency                 18 ms           12 ms    -33.3%
```

---

## Benchmark Types

### Network Benchmarks

Measures network performance within the cluster.

```bash
sloth-kubernetes benchmark run --type network
```

**Metrics measured:**
- **Pod-to-Pod Latency**: Round-trip time between pods on different nodes
- **DNS Resolution**: Time to resolve cluster DNS names
- **Throughput**: Network bandwidth between pods
- **CNI Performance**: Container Network Interface efficiency

**Reference values:**
- Pod-to-Pod Latency: < 1ms (excellent), < 5ms (good), < 10ms (acceptable)
- DNS Resolution: < 5ms (excellent), < 20ms (good), < 50ms (acceptable)

### Storage Benchmarks

Measures storage I/O performance.

```bash
sloth-kubernetes benchmark run --type storage
```

**Metrics measured:**
- **IOPS**: Input/Output Operations Per Second (4K random read/write)
- **Sequential I/O**: Large block sequential read/write throughput
- **Random I/O**: Small block random read/write latency
- **PVC Provisioning**: Time to provision a PersistentVolumeClaim

**Reference values:**
- IOPS: > 10000 (excellent), > 5000 (good), > 1000 (acceptable)
- Sequential throughput: > 500MB/s (excellent), > 200MB/s (good)

### CPU Benchmarks

Measures compute performance and scheduling efficiency.

```bash
sloth-kubernetes benchmark run --type cpu
```

**Metrics measured:**
- **CPU Efficiency**: Processing power utilization
- **Scheduling Latency**: Time for scheduler to place pods
- **API Server Response**: Kubernetes API responsiveness

**Reference values:**
- Scheduling Latency: < 100ms (excellent), < 500ms (good), < 1s (acceptable)
- API Server Response: < 50ms (excellent), < 200ms (good), < 500ms (acceptable)

### Memory Benchmarks

Measures memory performance and utilization.

```bash
sloth-kubernetes benchmark run --type memory
```

**Metrics measured:**
- **Memory Utilization**: Cluster-wide memory usage efficiency
- **Memory Bandwidth**: Read/write throughput to memory
- **etcd Memory Usage**: Memory consumed by etcd cluster

**Reference values:**
- Memory Utilization: 60-80% (optimal), < 60% (underutilized), > 90% (pressure)

---

## Output Formats

### Text (Default)

Human-readable detailed report with color-coded status.

```bash
sloth-kubernetes benchmark run --output text
```

### JSON

Machine-readable format for automation and storage.

```bash
sloth-kubernetes benchmark run --output json
```

Example JSON structure:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "cluster_name": "production",
  "results": [
    {
      "name": "Pod-to-Pod Latency",
      "category": "network",
      "score": 0.45,
      "unit": "ms",
      "status": "passed",
      "reference": 1.0,
      "details": "Measured across 3 node pairs"
    }
  ],
  "summary": {
    "overall_score": 85.2,
    "grade": "B+",
    "network_score": 88.0,
    "storage_score": 82.5,
    "cpu_score": 86.3,
    "memory_score": 84.0,
    "recommendations": [
      "Consider SSD storage for better IOPS"
    ]
  }
}
```

### Compact

Condensed summary view.

```bash
sloth-kubernetes benchmark run --output compact
```

---

## Examples

### Baseline Performance Check

Establish a performance baseline for your cluster:

```bash
# Run full benchmark suite and save
sloth-kubernetes benchmark run --save baseline-$(date +%Y%m%d).json

# View the saved report
sloth-kubernetes benchmark report baseline-20240115.json
```

### Performance Monitoring

Regular performance monitoring workflow:

```bash
# Weekly performance check
sloth-kubernetes benchmark run \
  --save weekly-$(date +%Y%m%d).json \
  --compare baseline.json
```

### Node-Specific Benchmarks

Benchmark specific nodes to identify issues:

```bash
# Benchmark only worker nodes
sloth-kubernetes benchmark run --nodes worker-1,worker-2,worker-3

# Benchmark a specific problematic node
sloth-kubernetes benchmark run --nodes worker-5 --verbose
```

### Pre-Production Validation

Validate cluster performance before production deployment:

```bash
# Run comprehensive benchmarks
sloth-kubernetes benchmark run --verbose --output text

# Quick sanity check
sloth-kubernetes benchmark quick
```

### CI/CD Integration

Automated performance testing in pipelines:

```bash
#!/bin/bash
# benchmark-check.sh

sloth-kubernetes benchmark run --output json --save results.json

# Extract overall score
SCORE=$(jq '.summary.overall_score' results.json)

# Fail if score below threshold
if (( $(echo "$SCORE < 70" | bc -l) )); then
  echo "Performance below threshold: $SCORE"
  exit 1
fi

echo "Performance check passed: $SCORE"
```

---

## Scoring System

### Overall Score

The overall score (0-100) is a weighted average of category scores:
- Network: 25%
- Storage: 25%
- CPU: 25%
- Memory: 25%

### Grades

| Score Range | Grade | Interpretation |
|-------------|-------|----------------|
| 90-100 | A | Excellent performance |
| 80-89 | B | Good performance |
| 70-79 | C | Acceptable performance |
| 60-69 | D | Below average, improvements recommended |
| 0-59 | F | Poor performance, action required |

### Status Indicators

- **[OK]** / **PASSED**: Metric within acceptable range
- **[WARN]**: Metric approaching limits
- **[FAIL]** / **FAILED**: Metric outside acceptable range

---

## Troubleshooting

### Benchmark Fails to Connect

If benchmarks fail to connect to the cluster:

```bash
# Verify kubeconfig
sloth-kubernetes benchmark run --kubeconfig ~/.kube/config

# Check cluster connectivity
sloth-kubernetes kubectl get nodes
```

### Low Network Scores

Common causes and solutions:
- **High latency**: Check CNI configuration, consider network policy optimization
- **Low throughput**: Verify MTU settings, check for bandwidth throttling
- **DNS issues**: Review CoreDNS configuration and resources

### Low Storage Scores

Common causes and solutions:
- **Low IOPS**: Consider SSD storage, check for noisy neighbors
- **Slow provisioning**: Review storage class settings, check CSI driver logs

### Inconsistent Results

For more reliable results:

```bash
# Run multiple times and compare
for i in 1 2 3; do
  sloth-kubernetes benchmark run --save run-$i.json
done

# Compare runs
sloth-kubernetes benchmark compare run-1.json run-3.json
```
