package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestROCmJSON(t *testing.T) {
	fixture := []byte(`{"card0":{"VRAM Total Memory (B)":"2147483648","VRAM Total Used Memory (B)":"1073741824","GPU use (%)":"75","Temperature (Sensor edge) (C)":"65.5"}}`)
	m, err := parseROCmJSON(fixture, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.GPUMemoryUsedMB != 1024 || m.GPUMemoryTotalMB != 2048 || m.GPUMemoryPercent != 50 || m.GPUUtilizationPercent != 75 || m.Source != "ROCmSMI" {
		t.Fatal(m)
	}
	if _, err := parseROCmJSON(fixture, 1); err == nil {
		t.Fatal("missing device accepted")
	}
	for _, invalid := range []string{`{}`, `{"card0":{}}`, `not JSON`} {
		if _, err := parseROCmJSON([]byte(invalid), 0); err == nil {
			t.Fatal("invalid metrics accepted")
		}
	}
}
func TestNVIDIACollectorCommand(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo version; else echo '1024, 2048, 80, 65'; fi\n"
	if err := os.WriteFile(filepath.Join(dir, "nvidia-smi"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	c := GetCollector("")
	m, err := c.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if c.Name() != "NVIDIASMI" || m.GPUMemoryPercent != 50 || m.Source != "NVIDIASMI" {
		t.Fatal(m)
	}
}

func TestSimulationScenariosAndIndependentProgression(t *testing.T) {
	oom := NewSimulatedCollector("gpu_oom")
	other := NewSimulatedCollector("gpu_oom")
	for _, percent := range []float64{20, 45, 70, 85, 91, 95, 98, 99, 99} {
		sample, err := oom.Collect()
		if err != nil || sample.Source != "Simulated" || sample.GPUMemoryPercent != percent || sample.GPUMemoryUsedMB != percent/100*sample.GPUMemoryTotalMB {
			t.Fatalf("unexpected demo progression: %+v %v", sample, err)
		}
	}
	first, _ := other.Collect()
	if first.GPUMemoryPercent != 20 {
		t.Fatal("jobs shared simulation state")
	}
	for _, scenario := range []string{"timeout", "successful", "default"} {
		collector := NewSimulatedCollector(scenario)
		for i := 0; i < 15; i++ {
			sample, _ := collector.Collect()
			if sample.Source != "Simulated" || sample.GPUMemoryPercent < 0 || sample.GPUMemoryPercent > 100 || sample.GPUUtilizationPercent < 0 || sample.GPUUtilizationPercent > 100 {
				t.Fatalf("invalid demo sample: %+v", sample)
			}
			if scenario == "timeout" && (sample.GPUMemoryPercent != 45 || sample.GPUUtilizationPercent >= 8) {
				t.Fatal("timeout should simulate stalled utilization")
			}
		}
	}
}
