package db

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestClientRunRegistrationIsUniquePerOwnerAndSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	ids := make(chan int64, 12)
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			id, _, err := d.CreateClientWorkload("training", "ML_JOB", "running", "alice", "client-run")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		}()
	}
	workers.Wait()
	close(ids)
	var first int64
	for id := range ids {
		if first != 0 && id != first {
			t.Fatal("registration retry created another workload")
		}
		first = id
	}
	other, created, err := d.CreateClientWorkload("training", "ML_JOB", "running", "bob", "client-run")
	if err != nil || !created || other == first {
		t.Fatal("client run IDs were shared across owners", err)
	}
	if _, _, err := d.CreateClientWorkload("different training", "ML_JOB", "running", "alice", "client-run"); !errors.Is(err, ErrClientRunConflict) {
		t.Fatal("client run ID reuse with a different name was accepted")
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, created, err := d.CreateClientWorkload("training", "ML_JOB", "running", "alice", "client-run")
	if err != nil || created || id != first {
		t.Fatal("restart lost registration identity", err)
	}
	list, err := d.GetWorkloads("alice")
	if err != nil || len(list) != 1 {
		t.Fatal("duplicate workloads", err)
	}
}

func TestOldDatabaseGainsClientRunIDsWithoutLosingWorkloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.CreateWorkload("existing training", "ML_JOB", "running"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("DROP INDEX workloads_client_run; ALTER TABLE workloads DROP COLUMN client_run_id"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.GetWorkload("1", "legacy"); err != nil {
		t.Fatal("migration lost an existing workload", err)
	}
	if _, created, err := d.CreateClientWorkload("new training", "ML_JOB", "running", "legacy", "new-run"); err != nil || !created {
		t.Fatal("migration did not enable retryable registration", err)
	}
}
