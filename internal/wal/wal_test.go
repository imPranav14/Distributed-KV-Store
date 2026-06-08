package wal

import (
	"path/filepath"
	"testing"

	"github.com/imPranav14/Distributed-KV-Store/internal/store"
)

func TestWAL_AppendReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.log")

	wal, err := Open(path)
	if err != nil {
		t.Fatalf("Open wal: %v", err)
	}
	defer wal.Close()

	store1 := store.New()
	walStore := NewWALStore(store1, wal)

	if err := walStore.Put("k", "v"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := walStore.Append("k", "x"); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if err := wal.Close(); err != nil {
		t.Fatalf("Close wal: %v", err)
	}

	wal2, err := Open(path)
	if err != nil {
		t.Fatalf("Open wal2: %v", err)
	}
	defer wal2.Close()

	store2 := store.New()
	if err := wal2.ReplayInto(store2); err != nil {
		t.Fatalf("ReplayInto failed: %v", err)
	}

	value, ok := store2.Get("k")
	if !ok {
		t.Fatalf("expected key present after replay")
	}
	if value != "vx" {
		t.Fatalf("expected value vx after replay, got %q", value)
	}
}

func TestWAL_ReplayToleratesTruncatedFinalEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.log")

	wal, err := Open(path)
	if err != nil {
		t.Fatalf("Open wal: %v", err)
	}
	defer wal.Close()

	if err := wal.Append(&Entry{Type: OpTypePut, Key: "k", Value: "v"}); err != nil {
		t.Fatalf("Append first entry: %v", err)
	}

	// Simulate a crash by writing an incomplete second record directly.
	wal.mu.Lock()
	if _, err := wal.file.Write([]byte{0, 0, 0, 0, 0, 0, 0, 5}); err != nil {
		wal.mu.Unlock()
		t.Fatalf("write partial length: %v", err)
	}
	if _, err := wal.file.Write([]byte{1, 2}); err != nil {
		wal.mu.Unlock()
		t.Fatalf("write partial data: %v", err)
	}
	wal.mu.Unlock()

	if err := wal.Close(); err != nil {
		t.Fatalf("Close wal: %v", err)
	}

	wal2, err := Open(path)
	if err != nil {
		t.Fatalf("Open wal2: %v", err)
	}
	defer wal2.Close()

	store2 := store.New()
	if err := wal2.ReplayInto(store2); err != nil {
		t.Fatalf("ReplayInto failed: %v", err)
	}

	value, ok := store2.Get("k")
	if !ok {
		t.Fatalf("expected key present after replay")
	}
	if value != "v" {
		t.Fatalf("expected value v after truncated replay, got %q", value)
	}
}
