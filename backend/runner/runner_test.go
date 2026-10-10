package runner

import (
	"bytes"
	"context"
	"crashlens/db"
	"crashlens/metrics"
	"fmt"
	"os"
	"os/exec"
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
	return &metrics.GPUMetric{Source: "NVIDIASMI", Timestamp: time.Now(), GPUMemoryUsedMB: 1024, GPUMemoryTotalMB: 2048, GPUMemoryPercent: 50, GPUUtilizationPercent: 90}, nil
}
func TestRunnerUsesRealCollector(t *testing.T) {
	s := snapshotFromCollector(fakeCollector{}, metrics.NewSimulatedCollector("gpu_oom"))
	if s.Source != "NVIDIASMI" || s.GPUMemoryUsed != 1024 || s.GPUUtilization != 90 {
		t.Fatal(s)
	}
}

func TestLongLogLineIsNotTruncated(t *testing.T) {
	text := strings.Repeat("x", 100000) + "\nRuntimeError: CUDA out of memory\n"
	var buffer bytes.Buffer
	var mu sync.Mutex
	writer := &lockedLogWriter{buffer: &buffer, mutex: &mu}
	if n, err := writer.Write([]byte(text)); err != nil || n != len(text) {
		t.Fatalf("write failed: n=%d err=%v", n, err)
	}
	if buffer.String() != text {
		t.Fatal("logs were truncated")
	}
}

func TestLogWriterKeepsBoundedTail(t *testing.T) {
	var buffer bytes.Buffer
	var mu sync.Mutex
	writer := &lockedLogWriter{buffer: &buffer, mutex: &mu}
	text := strings.Repeat("x", 300*1024) + "\nRuntimeError: CUDA out of memory\n"
	n, err := writer.Write([]byte(text))
	if err != nil || n != len(text) {
		t.Fatalf("write failed: n=%d err=%v", n, err)
	}
	if buffer.String() != text[len(text)-256*1024:] {
		t.Fatal("writer did not retain bounded log tail")
	}
}

func TestLogWriterConcurrentStreams(t *testing.T) {
	var buffer bytes.Buffer
	var mu sync.Mutex
	writer := &lockedLogWriter{buffer: &buffer, mutex: &mu}
	var wg sync.WaitGroup
	for _, text := range []string{"stdout\n", "stderr\n"} {
		wg.Add(1)
		go func(text string) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				writer.Write([]byte(text))
			}
		}(text)
	}
	wg.Wait()
	logs := buffer.String()
	if strings.Count(logs, "stdout\n") != 1000 || strings.Count(logs, "stderr\n") != 1000 {
		t.Fatal("concurrent log writes lost or mixed chunks")
	}
}

func pythonScript(t *testing.T, content string) string {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	path := filepath.Join(t.TempDir(), "job.py")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestDeadlineFinalizesJob(t *testing.T) {
	d, err := db.New(filepath.Join(t.TempDir(), "deadline.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	script := pythonScript(t, "import time\nprint('CUDA out of memory in previous attempt', flush=True)\ntime.sleep(30)\n")
	id, _ := d.CreateWorkload("deadline", "ML_JOB", "pending")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = RunPythonJobContext(ctx, script, int(id), d)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("timeout did not bound execution")
	}
	w, _ := d.GetWorkload(fmt.Sprint(id))
	if w.Status != "failed" || w.FailureType == nil || *w.FailureType != "TIMEOUT" {
		t.Fatalf("timeout not recorded: %+v", w)
	}
}
func TestBoundedQueue(t *testing.T) {
	d, err := db.New(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	script := pythonScript(t, "import time\ntime.sleep(30)\n")
	manager := NewManager(d, 1, 1, time.Minute)
	defer manager.Close()
	first, err := manager.Submit("first", "ML_JOB", script)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		w, _ := d.GetWorkload(fmt.Sprint(first))
		if w.Status == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	second, err := manager.Submit("second", "ML_JOB", script)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Submit("third", "ML_JOB", script); err != ErrQueueFull {
		t.Fatalf("queue not bounded: %v", err)
	}
	queued, _ := d.GetWorkload(fmt.Sprint(second))
	if queued.Status != "pending" {
		t.Fatal("concurrency exceeded")
	}
}
func TestTelemetryVisibleBeforeCompletion(t *testing.T) {
	d, err := db.New(filepath.Join(t.TempDir(), "live.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	script := pythonScript(t, "import time\nprint('live progress', flush=True)\ntime.sleep(3)\n")
	id, _ := d.CreateWorkload("live", "ML_JOB", "pending")
	done := make(chan error, 1)
	go func() { _, err := RunPythonJob(script, int(id), d); done <- err }()
	found := false
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		w, _ := d.GetWorkload(fmt.Sprint(id))
		if w.Status == "running" && w.JobLogs != nil && strings.Contains(*w.JobLogs, "live progress") && w.GPUMetrics != nil && *w.GPUMetrics != "[]" {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("no telemetry before process exit")
	}
}

type failingCollector struct{ fakeCollector }

func (failingCollector) Collect() (*metrics.GPUMetric, error) {
	return nil, fmt.Errorf("GPU unavailable")
}
func TestRunnerFallbackUsesSharedSimulation(t *testing.T) {
	fallback := metrics.NewSimulatedCollector("gpu_oom")
	for _, percent := range []float64{20, 45, 70} {
		sample := snapshotFromCollector(failingCollector{}, fallback)
		if sample.Source != "Simulated" || sample.GPUMemoryPercent != percent {
			t.Fatalf("unexpected fallback: %+v", sample)
		}
	}
}
