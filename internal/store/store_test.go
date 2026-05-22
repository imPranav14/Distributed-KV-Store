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

// TestPut_ThenGet_RoundTrip asserts the core safety invariant:
// immediately after Put(k, v), Get(k) returns (v, true). This is the
// template every later state-machine invariant builds on.
//
// Refactored from M1.3's TestGet_PresentKey_ReturnsValueAndOK, which
// poked s.kv directly because Put did not yet exist.
func TestPut_ThenGet_RoundTrip(t *testing.T) {
	s := New()

	s.Put("k", "v")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after Put, got ok=false")
	}
	if v != "v" {
		t.Errorf("expected value %q, got %q", "v", v)
	}
}

// TestPut_EmptyString_IsObservable asserts that Put(k, "") is observably
// distinct from never writing k: Get must report ok=true. If this test
// ever fails, Put("k", "") has become indistinguishable from "k was never
// written" and WAL replay / snapshot restore will silently corrupt state.
//
// Refactored from M1.3's TestGet_PresentKey_EmptyString_DistinctFromAbsent.
func TestPut_EmptyString_IsObservable(t *testing.T) {
	s := New()

	s.Put("empty", "")
	v, ok := s.Get("empty")

	if !ok {
		t.Errorf("present-with-empty-value must be distinguishable from absent: expected ok=true, got ok=false")
	}
	if v != "" {
		t.Errorf("expected empty-string value, got %q", v)
	}
}

// TestPut_OverwritesExistingValue asserts that a second Put on the same
// key replaces the value rather than appending or erroring.
func TestPut_OverwritesExistingValue(t *testing.T) {
	s := New()

	s.Put("k", "first")
	s.Put("k", "second")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after overwrite, got ok=false")
	}
	if v != "second" {
		t.Errorf("expected overwritten value %q, got %q", "second", v)
	}
}

// TestPut_MultipleKeys_AreIndependent asserts that writes to one key do
// not affect any other key. Trivial for a map[string]string today; this
// test exists as a regression guard for later layers (e.g., a buggy WAL
// or snapshot encoder that conflates keys).
func TestPut_MultipleKeys_AreIndependent(t *testing.T) {
	s := New()

	s.Put("a", "1")
	s.Put("b", "2")

	if v, ok := s.Get("a"); !ok || v != "1" {
		t.Errorf("Get(a) = (%q, %v), want (%q, true)", v, ok, "1")
	}
	if v, ok := s.Get("b"); !ok || v != "2" {
		t.Errorf("Get(b) = (%q, %v), want (%q, true)", v, ok, "2")
	}
}

// TestPut_RepeatedSameValue_IsIdempotent names the property that makes Put
// safe for client retry: applying the same Put twice yields the same
// observable state as applying it once. Append (M1.5) deliberately does
// not have this property; that contrast is the motivation for M2's
// idempotency machinery.
func TestPut_RepeatedSameValue_IsIdempotent(t *testing.T) {
	s := New()

	s.Put("k", "v")
	s.Put("k", "v")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after repeated Put, got ok=false")
	}
	if v != "v" {
		t.Errorf("expected value %q after repeated Put, got %q", "v", v)
	}
}

// TestAppend_OnAbsentKey_CreatesIt asserts the create-if-missing half of
// Append's semantics. Go's map zero value (empty string) makes this fall
// out of the implementation without a branch.
func TestAppend_OnAbsentKey_CreatesIt(t *testing.T) {
	s := New()

	s.Append("k", "v")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after Append on absent key, got ok=false")
	}
	if v != "v" {
		t.Errorf("expected value %q after Append on absent key, got %q", "v", v)
	}
}

// TestAppend_OnExistingKey_Concatenates asserts the concatenation half of
// Append's semantics.
func TestAppend_OnExistingKey_Concatenates(t *testing.T) {
	s := New()

	s.Put("k", "ab")
	s.Append("k", "cd")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after Append, got ok=false")
	}
	if v != "abcd" {
		t.Errorf("expected concatenated value %q, got %q", "abcd", v)
	}
}

// TestAppend_EmptyValueOnExisting_IsNoOp asserts that Append(k, "") on an
// existing key leaves the value unchanged but still observable as present.
func TestAppend_EmptyValueOnExisting_IsNoOp(t *testing.T) {
	s := New()

	s.Put("k", "abc")
	s.Append("k", "")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after Append(\"\"), got ok=false")
	}
	if v != "abc" {
		t.Errorf("expected unchanged value %q after empty Append, got %q", "abc", v)
	}
}

// TestAppend_Sequence_AccumulatesAllValues asserts that successive Appends
// build up the expected concatenation. This is the accumulator pattern that
// distributed clients (e.g., a log shipper) rely on.
func TestAppend_Sequence_AccumulatesAllValues(t *testing.T) {
	s := New()

	s.Append("k", "a")
	s.Append("k", "b")
	s.Append("k", "c")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after Append sequence, got ok=false")
	}
	if v != "abc" {
		t.Errorf("expected accumulated value %q, got %q", "abc", v)
	}
}

// TestAppend_IsNotIdempotent is the negative twin of
// TestPut_RepeatedSameValue_IsIdempotent and the motivating example for M2.
// Applying Append twice produces a state observably different from applying
// it once. If this test ever starts failing, Append's semantics have been
// mistakenly weakened — and the case for M2's request-ID dedup machinery
// goes with it.
func TestAppend_IsNotIdempotent(t *testing.T) {
	s := New()

	s.Append("k", "x")
	s.Append("k", "x")
	v, ok := s.Get("k")

	if !ok {
		t.Errorf("expected ok=true after two Appends, got ok=false")
	}
	if v != "xx" {
		t.Errorf("Append is meant to be NON-idempotent: expected %q after two Appends, got %q", "xx", v)
	}
}
