package db

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestOwnershipMigrationAndKeysSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	d, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := d.CreateWorkload("old workload", "ML_JOB", "running")
	if _, err := d.Exec("DROP INDEX workloads_owner; ALTER TABLE workloads DROP COLUMN owner_id"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.GetWorkload(fmt.Sprint(id), "legacy"); err != nil {
		t.Fatal("migration lost legacy ownership", err)
	}
	key, token, err := d.IssueAPIKey("Alice", "")
	if err != nil {
		t.Fatal(err)
	}
	alice, _ := d.CreateWorkload("Alice", "ML_JOB", "pending", key.OwnerID)
	d.Close()
	d, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if owner, err := d.ResolveAPIKey(token); err != nil || owner != key.OwnerID {
		t.Fatal("key lost on restart")
	}
	if _, err := d.GetWorkload(fmt.Sprint(alice), key.OwnerID); err != nil {
		t.Fatal("ownership lost on restart", err)
	}
	if _, err := d.GetWorkload(fmt.Sprint(id), key.OwnerID); err == nil {
		t.Fatal("legacy workload visible to new user")
	}
	if _, err := d.RevokeAPIKey(key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ResolveAPIKey(token); err == nil {
		t.Fatal("revoked key accepted")
	}
}
