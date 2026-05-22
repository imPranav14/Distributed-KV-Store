package store

import "testing"

// TestNew_ReturnsUsableEmptyStore checks the construction invariant:
// after New, the Store exists, its backing map is allocated (so writes
// would not panic), and it contains no entries.
//
// This test uses same-package access to inspect the unexported kv field
// because no public read method exists yet; once Get lands in M1.3, the
// test can be re-expressed through the public API.
func TestNew_ReturnsUsableEmptyStore(t *testing.T) {
	s := New()

	if s == nil {
		t.Fatal("New() returned nil *Store")
	}
	if s.kv == nil {
		t.Fatal("New() returned Store with nil map; writes would panic")
	}
	if got := len(s.kv); got != 0 {
		t.Errorf("expected empty store, got %d entries", got)
	}
}
