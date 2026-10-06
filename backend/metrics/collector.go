package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GPUMetric represents a single point-in-time GPU measurement
type GPUMetric struct {
	Timestamp             time.Time `json:"timestamp"`
	GPUMemoryUsedMB       float64   `json:"gpu_memory_used_mb"`
	GPUMemoryTotalMB      float64   `json:"gpu_memory_total_mb"`
	GPUMemoryPercent      float64   `json:"gpu_memory_percent"`
	GPUUtilizationPercent float64   `json:"gpu_utilization_percent"`
	Source                string    `json:"source"`
	TemperatureCelsius    int       `json:"temperature_celsius"`
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
	_, err := smiOutput("rocm-smi", "--version")
	return err == nil
}

func (c *ROCmSMICollector) Name() string {
	return "ROCmSMI"
}

func (c *ROCmSMICollector) Collect() (*GPUMetric, error) {
	output, err := smiOutput("rocm-smi", "--showmeminfo", "vram", "--showuse", "--showtemp", "--json")
	if err != nil {
		return nil, fmt.Errorf("rocm-smi: %w", err)
	}
	return parseROCmJSON(output, c.deviceID)
}

// Keys follow ROCm/rocm_smi_lib python_smi_tools/rocm_smi.py.
func parseROCmJSON(output []byte, deviceID int) (*GPUMetric, error) {
	var cards map[string]map[string]interface{}
	if err := json.Unmarshal(output, &cards); err != nil {
		return nil, err
	}
	card, ok := cards[fmt.Sprintf("card%d", deviceID)]
	if !ok {
		return nil, fmt.Errorf("GPU %d missing from rocm-smi output", deviceID)
	}
	number := func(key string) (float64, error) {
		value, ok := card[key]
		if !ok {
			return 0, fmt.Errorf("missing %s", key)
		}
		n, err := strconv.ParseFloat(fmt.Sprint(value), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, fmt.Errorf("invalid %s", key)
		}
		return n, nil
	}
	total, err := number("VRAM Total Memory (B)")
	if err != nil || total <= 0 {
		return nil, fmt.Errorf("invalid VRAM total")
	}
	used, err := number("VRAM Total Used Memory (B)")
	if err != nil || used < 0 || used > total {
		return nil, fmt.Errorf("invalid VRAM used")
	}
	util, err := number("GPU use (%)")
	if err != nil || util < 0 || util > 100 {
		return nil, fmt.Errorf("invalid GPU utilization")
	}
	temp, err := number("Temperature (Sensor edge) (C)")
	if err != nil {
		temp, err = number("Temperature (Sensor junction) (C)")
	}
	if err != nil {
		return nil, err
	}
	return &GPUMetric{Timestamp: time.Now().UTC(), GPUMemoryUsedMB: used / (1024 * 1024),
		GPUMemoryTotalMB: total / (1024 * 1024), GPUMemoryPercent: used / total * 100,
		GPUUtilizationPercent: util, TemperatureCelsius: int(temp), Source: "ROCmSMI"}, nil
}

func smiOutput(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// NVIDIASMICollector collects real metrics from NVIDIA GPUs using nvidia-smi
type NVIDIASMICollector struct {
	deviceID int
}

// NewNVIDIASMICollector creates a collector for NVIDIA CUDA GPUs
func NewNVIDIASMICollector(deviceID int) *NVIDIASMICollector {
	return &NVIDIASMICollector{deviceID: deviceID}
}

func (c *NVIDIASMICollector) IsAvailable() bool {
	// Check if nvidia-smi is available
	_, err := smiOutput("nvidia-smi", "--version")
	return err == nil
}

func (c *NVIDIASMICollector) Name() string {
	return "NVIDIASMI"
}

func (c *NVIDIASMICollector) Collect() (*GPUMetric, error) {
	// Use nvidia-smi with CSV format for easy parsing
	// Query: memory.used, memory.total, utilization.gpu, temperature.gpu
	output, err := smiOutput("nvidia-smi",
		"--query-gpu=memory.used,memory.total,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits",
		fmt.Sprintf("--id=%d", c.deviceID))

	if err != nil {
		return nil, fmt.Errorf("failed to run nvidia-smi: %w", err)
	}

	// Parse CSV output: "memUsed, memTotal, utilization, temperature"
	// Example: "1024, 16384, 85, 65"
	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, ",")

	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected nvidia-smi output format: %s", line)
	}

	memUsed, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory used: %w", err)
	}

	memTotal, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory total: %w", err)
	}

	utilization, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse utilization: %w", err)
	}

	temperature, err := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse temperature: %w", err)
	}

	memPercent := 0.0
	if memTotal > 0 {
		memPercent = (memUsed / memTotal) * 100
	}

	return &GPUMetric{
		Timestamp:             time.Now(),
		Source:                c.Name(),
		GPUMemoryUsedMB:       memUsed,
		GPUMemoryTotalMB:      memTotal,
		GPUMemoryPercent:      memPercent,
		GPUUtilizationPercent: utilization,
		TemperatureCelsius:    int(temperature),
	}, nil
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
	return "Simulated"
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
		Source:                c.Name(),
		GPUMemoryUsedMB:       memoryUsed,
		GPUMemoryTotalMB:      c.totalMemoryMB,
		GPUMemoryPercent:      memoryPercent,
		GPUUtilizationPercent: utilization,
		TemperatureCelsius:    temperature,
	}, nil
}

// GetCollector returns the appropriate collector based on environment
// It auto-detects GPU type: NVIDIA (nvidia-smi), AMD (rocm-smi), or simulated
func GetCollector(scenario string) MetricCollector {
	// Try NVIDIA collector first (most common)
	nvidiaCollector := NewNVIDIASMICollector(0)
	if nvidiaCollector.IsAvailable() {
		return nvidiaCollector
	}

	// Try AMD ROCm collector
	rocmCollector := NewROCmSMICollector(0)
	if rocmCollector.IsAvailable() {
		return rocmCollector
	}

	// Fallback to simulated collector for demo/testing
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
