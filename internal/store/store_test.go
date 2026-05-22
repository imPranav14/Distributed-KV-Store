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

// TestGet_AbsentKey_ReturnsNotOK asserts the absent-key half of the
// comma-ok contract: missing keys return ("", false).
func TestGet_AbsentKey_ReturnsNotOK(t *testing.T) {
	s := New()

	v, ok := s.Get("missing")

	if ok {
		t.Errorf("expected ok=false for absent key, got ok=true (value=%q)", v)
	}
	if v != "" {
		t.Errorf("expected zero value for absent key, got %q", v)
	}
}

// TestGet_PresentKey_ReturnsValueAndOK asserts the present-key half: a key
// written into the map is returned with ok=true and the correct value.
// The test bypasses the (not-yet-implemented) Put by writing directly to
// the unexported kv field; M1.4 will re-express this test through Put.
func TestGet_PresentKey_ReturnsValueAndOK(t *testing.T) {
	s := New()
	s.kv["k"] = "v"

	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true for present key, got ok=false")
	}
	if v != "v" {
		t.Errorf("expected value %q, got %q", "v", v)
	}
}

// TestGet_PresentKey_EmptyString_DistinctFromAbsent is the pedagogical core
// of M1.3: a key present with an empty-string value must be observably
// distinct from an absent key. If this test ever fails, Put("k", "") has
// become indistinguishable from "k was never written" and WAL replay /
// snapshot restore will silently corrupt state.
func TestGet_PresentKey_EmptyString_DistinctFromAbsent(t *testing.T) {
	s := New()
	s.kv["empty"] = ""

	v, ok := s.Get("empty")

	if !ok {
		t.Errorf("present-with-empty-value must be distinguishable from absent: expected ok=true, got ok=false")
	}
	if v != "" {
		t.Errorf("expected empty-string value, got %q", v)
	}
}
