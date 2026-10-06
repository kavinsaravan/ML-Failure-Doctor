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

func TestRecoveryOnlyFinalizesManagedJobs(t *testing.T) {
	d, err := New(filepath.Join(t.TempDir(), "recover.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	external, err := d.CreateWorkload("SDK", "ML_JOB", "running")
	if err != nil {
		t.Fatal(err)
	}
	managed, err := d.CreateManagedWorkload("demo", "ML_JOB")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.RecoverManagedJobs("restart"); err != nil {
		t.Fatal(err)
	}
	sdk, err := d.GetWorkload(fmt.Sprint(external))
	if err != nil || sdk.Status != "running" {
		t.Fatal("external training was killed")
	}
	job, err := d.GetWorkload(fmt.Sprint(managed))
	if err != nil || job.Status != "failed" || job.FinishedAt == nil {
		t.Fatal("managed work stranded")
	}
	if err := d.UpdateTelemetry(int(managed), "late", "[]", 10); err != nil {
		t.Fatal(err)
	}
	after, _ := d.GetWorkload(fmt.Sprint(managed))
	if *after.JobLogs == "late" {
		t.Fatal("late telemetry changed a completed job")
	}
	after.Status = "running"
	if err := d.UpdateWorkloadIfStatus(fmt.Sprint(managed), after, "pending"); err != ErrWorkloadChanged {
		t.Fatal("stale write accepted")
	}
}

func TestExistingDatabaseMigratesManagedFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("ALTER TABLE workloads DROP COLUMN runner_managed"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.CreateManagedWorkload("migrated", "ML_JOB"); err != nil {
		t.Fatal(err)
	}
}
