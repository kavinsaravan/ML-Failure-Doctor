package db

import (
	"fmt"
	"path/filepath"
	"testing"
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
	id, err := d.CreateWorkload("SDK", "ML_JOB", "running")
	if err != nil {
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
}
