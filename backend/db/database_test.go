package db

import (
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
