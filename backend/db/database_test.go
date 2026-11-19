package db

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestStatsDatabaseError(t *testing.T) {
	d, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if _, err := d.GetStats(); err == nil {
		t.Fatal("expected database error")
	}
}

// Existing databases may retain the obsolete runner flag. Opening them must
// preserve SDK runs and allow normal lifecycle updates without a data migration.
func TestExistingDatabasePreservesSDKRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("ALTER TABLE workloads ADD COLUMN runner_managed INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("ALTER TABLE workloads ADD COLUMN checkpoint_state TEXT"); err != nil {
		t.Fatal(err)
	}
	id, err := d.CreateWorkload("SDK", "ML_JOB", "running")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("UPDATE workloads SET checkpoint_state='legacy metadata' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	workload, err := d.GetWorkload(fmt.Sprint(id))
	if err != nil || workload.Status != "running" {
		t.Fatalf("SDK run changed on reopen: %+v %v", workload, err)
	}
	workload.Status = "succeeded"
	if err := d.UpdateWorkloadIfStatus(fmt.Sprint(id), workload, "running"); err != nil {
		t.Fatal(err)
	}
	workload.Status = "failed"
	if err := d.UpdateWorkloadIfStatus(fmt.Sprint(id), workload, "running"); err != ErrWorkloadChanged {
		t.Fatal("stale write accepted")
	}
	persisted, err := d.GetWorkload(fmt.Sprint(id))
	if err != nil || persisted.Status != "succeeded" || persisted.FinishedAt == nil {
		t.Fatalf("completed run changed: %+v %v", persisted, err)
	}
	var legacy string
	if err := d.QueryRow("SELECT checkpoint_state FROM workloads WHERE id=?", id).Scan(&legacy); err != nil || legacy != "legacy metadata" {
		t.Fatal("existing checkpoint metadata was altered", err)
	}
}

func TestHeartbeatMigrationPreservesRunningJobs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-heartbeat.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.CreateWorkload("legacy training", "ML_JOB", "running")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("UPDATE workloads SET started_at=? WHERE id=?", time.Now().Add(-3*HeartbeatTimeout), id); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("ALTER TABLE workloads DROP COLUMN last_heartbeat_at"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	w, err := d.GetWorkload(fmt.Sprint(id))
	if err != nil || !w.HeartbeatStale || w.Status != "running" || w.LastHeartbeatAt != nil {
		t.Fatalf("legacy run was lost or changed: %+v %v", w, err)
	}
	w.Status = "failed"
	logs := "ModuleNotFoundError: missing package"
	w.JobLogs = &logs
	if err := d.UpdateWorkload(fmt.Sprint(id), w); err != nil {
		t.Fatal(err)
	}
	w, err = d.GetWorkload(fmt.Sprint(id))
	if err != nil || w.HeartbeatStale || w.LastHeartbeatAt == nil || w.Status != "failed" {
		t.Fatalf("legacy final upload failed: %+v %v", w, err)
	}
}
