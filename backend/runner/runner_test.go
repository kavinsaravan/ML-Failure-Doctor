package runner

import (
	"bytes"
	"crashlens/db"
	"crashlens/metrics"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMissingPythonIsRecordedAsFailed(t *testing.T) {
	d, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, err := d.CreateWorkload("launch", "ML_JOB", "pending")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := RunPythonJob("missing.py", int(id), d); err == nil {
		t.Fatal("expected launch error")
	}
	w, err := d.GetWorkload(fmt.Sprint(id))
	if err != nil {
		t.Fatal(err)
	}
	if w.Status != "failed" || w.FinishedAt == nil || w.ExitCode == nil || *w.ExitCode != 1 || w.JobLogs == nil {
		t.Fatalf("launch failure not finalized: %+v", w)
	}
}

type fakeCollector struct{}

func (fakeCollector) Name() string      { return "NVIDIASMI" }
func (fakeCollector) IsAvailable() bool { return true }
func (fakeCollector) Collect() (*metrics.GPUMetric, error) {
	return &metrics.GPUMetric{Timestamp: time.Now(), GPUMemoryUsedMB: 1024, GPUMemoryTotalMB: 2048, GPUMemoryPercent: 50, GPUUtilizationPercent: 90}, nil
}
func TestRunnerUsesRealCollector(t *testing.T) {
	s := snapshotFromCollector(fakeCollector{}, "gpu_oom", 0)
	if s.Source != "NVIDIASMI" || s.GPUMemoryUsed != 1024 || s.GPUUtilization != 90 {
		t.Fatal(s)
	}
}

func TestLongLogLineIsNotTruncated(t *testing.T) {
	text := strings.Repeat("x", 100000) + "\nRuntimeError: CUDA out of memory\n"
	var buffer bytes.Buffer
	var mu sync.Mutex
	streamOutput(strings.NewReader(text), &buffer, &mu, "test")
	if buffer.String() != text {
		t.Fatal("logs were truncated")
	}
}
