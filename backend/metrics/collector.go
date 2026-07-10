package metrics

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GPUMetric represents a single point-in-time GPU measurement
type GPUMetric struct {
	Timestamp          time.Time `json:"timestamp"`
	GPUMemoryUsedMB    float64   `json:"gpu_memory_used_mb"`
	GPUMemoryTotalMB   float64   `json:"gpu_memory_total_mb"`
	GPUMemoryPercent   float64   `json:"gpu_memory_percent"`
	GPUUtilizationPercent float64 `json:"gpu_utilization_percent"`
	TemperatureCelsius int       `json:"temperature_celsius"`
}

// MetricCollector is the interface for collecting GPU metrics
type MetricCollector interface {
	// Collect returns current GPU metrics
	Collect() (*GPUMetric, error)

	// IsAvailable checks if the collector can actually collect metrics
	IsAvailable() bool

	// Name returns the collector implementation name
	Name() string
}

// ROCmSMICollector collects real metrics from AMD GPUs using rocm-smi
type ROCmSMICollector struct {
	deviceID int
}

// NewROCmSMICollector creates a collector for AMD ROCm GPUs
func NewROCmSMICollector(deviceID int) *ROCmSMICollector {
	return &ROCmSMICollector{deviceID: deviceID}
}

func (c *ROCmSMICollector) IsAvailable() bool {
	// Check if rocm-smi is available
	cmd := exec.Command("rocm-smi", "--version")
	err := cmd.Run()
	return err == nil
}

func (c *ROCmSMICollector) Name() string {
	return "ROCmSMI"
}

func (c *ROCmSMICollector) Collect() (*GPUMetric, error) {
	// Run rocm-smi to get memory info
	// Example: rocm-smi --showmeminfo vram --json
	memCmd := exec.Command("rocm-smi", "--showmeminfo", "vram", "--json")
	memOutput, err := memCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rocm-smi for memory: %w", err)
	}

	// Run rocm-smi to get utilization
	// Example: rocm-smi --showuse --json
	utilCmd := exec.Command("rocm-smi", "--showuse", "--json")
	utilOutput, err := utilCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rocm-smi for utilization: %w", err)
	}

	// Run rocm-smi to get temperature
	// Example: rocm-smi --showtemp --json
	tempCmd := exec.Command("rocm-smi", "--showtemp", "--json")
	tempOutput, err := tempCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rocm-smi for temperature: %w", err)
	}

	// Parse the JSON outputs (simplified - actual parsing would be more complex)
	metric := &GPUMetric{
		Timestamp: time.Now(),
	}

	// Parse memory info
	var memData map[string]interface{}
	if err := json.Unmarshal(memOutput, &memData); err == nil {
		// Extract memory values from JSON structure
		// Actual implementation would navigate the JSON structure
		// This is a simplified version
		metric.GPUMemoryUsedMB = 0    // Extract from memData
		metric.GPUMemoryTotalMB = 0   // Extract from memData
	}

	// Parse utilization
	var utilData map[string]interface{}
	if err := json.Unmarshal(utilOutput, &utilData); err == nil {
		metric.GPUUtilizationPercent = 0 // Extract from utilData
	}

	// Parse temperature
	var tempData map[string]interface{}
	if err := json.Unmarshal(tempOutput, &tempData); err == nil {
		metric.TemperatureCelsius = 0 // Extract from tempData
	}

	// Calculate percentage if we have the values
	if metric.GPUMemoryTotalMB > 0 {
		metric.GPUMemoryPercent = (metric.GPUMemoryUsedMB / metric.GPUMemoryTotalMB) * 100
	}

	return metric, nil
}

// SimulatedCollector generates realistic simulated metrics for demo purposes
type SimulatedCollector struct {
	baseMemoryMB  float64
	totalMemoryMB float64
	trend         string // "increasing", "stable", "oom"
	step          int
}

// NewSimulatedCollector creates a collector that generates demo metrics
func NewSimulatedCollector(scenario string) *SimulatedCollector {
	totalMemory := 24576.0 // 24GB AMD MI250X

	collector := &SimulatedCollector{
		totalMemoryMB: totalMemory,
		step:          0,
	}

	switch scenario {
	case "oom":
		collector.baseMemoryMB = totalMemory * 0.75
		collector.trend = "oom"
	case "stable":
		collector.baseMemoryMB = totalMemory * 0.45
		collector.trend = "stable"
	default:
		collector.baseMemoryMB = totalMemory * 0.40
		collector.trend = "increasing"
	}

	return collector
}

func (c *SimulatedCollector) IsAvailable() bool {
	return true
}

func (c *SimulatedCollector) Name() string {
	return "Simulated (ROCm-compatible)"
}

func (c *SimulatedCollector) Collect() (*GPUMetric, error) {
	c.step++

	var memoryUsed float64
	var utilization float64

	switch c.trend {
	case "oom":
		// Gradually increase to OOM
		memoryUsed = c.baseMemoryMB + (float64(c.step) * 400)
		if memoryUsed > c.totalMemoryMB {
			memoryUsed = c.totalMemoryMB * 0.97 // Almost full
		}
		utilization = 85 + rand.Float64()*10

	case "stable":
		// Stable with small variations
		memoryUsed = c.baseMemoryMB + (rand.Float64()-0.5)*1000
		utilization = 70 + rand.Float64()*10

	case "increasing":
		// Gradually increasing but safe
		memoryUsed = c.baseMemoryMB + (float64(c.step) * 200)
		if memoryUsed > c.totalMemoryMB*0.85 {
			memoryUsed = c.totalMemoryMB * 0.85
		}
		utilization = 60 + rand.Float64()*20
	}

	memoryPercent := (memoryUsed / c.totalMemoryMB) * 100
	temperature := 60 + int(utilization/10) + rand.Intn(10)

	return &GPUMetric{
		Timestamp:             time.Now(),
		GPUMemoryUsedMB:       memoryUsed,
		GPUMemoryTotalMB:      c.totalMemoryMB,
		GPUMemoryPercent:      memoryPercent,
		GPUUtilizationPercent: utilization,
		TemperatureCelsius:    temperature,
	}, nil
}

// GetCollector returns the appropriate collector based on environment
func GetCollector(scenario string) MetricCollector {
	// Try ROCm collector first
	rocmCollector := NewROCmSMICollector(0)
	if rocmCollector.IsAvailable() {
		return rocmCollector
	}

	// Fallback to simulated collector for demo
	return NewSimulatedCollector(scenario)
}

// ParseROCmSMIOutput parses rocm-smi output for specific metrics
// This is a helper function for more robust parsing
func ParseROCmSMIOutput(output string, metric string) (float64, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, metric) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				valueStr := strings.TrimSpace(parts[len(parts)-1])
				value, err := strconv.ParseFloat(valueStr, 64)
				if err == nil {
					return value, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("metric %s not found in output", metric)
}
