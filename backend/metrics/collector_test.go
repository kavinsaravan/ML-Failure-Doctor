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
