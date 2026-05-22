package store

// Store is the in-memory key-value state machine.
//
// Invariants:
//   - (Safety) After New, kv is non-nil. All methods assume this; callers
//     must construct a Store via New, never via the zero value Store{}.
//   - (Safety) Store performs no I/O, no time reads, and no random number
//     generation, so its behavior is a pure function of (state, input).
//
// Concurrency: Store is not safe for concurrent use in Milestone 1. A
// sync.RWMutex will be added in Milestone 3 when a gRPC handler becomes
// the first concurrent caller.
type Store struct {
	kv map[string]string
}

// New returns a freshly-initialized, empty Store ready for use.
//
// We return *Store (not Store) so that callers share one logical store by
// reference; copying a Store value would share the underlying map header
// and cause aliasing bugs.
func New() *Store {
	return &Store{
		kv: make(map[string]string),
	}
}

// Get returns the value associated with key and a boolean indicating whether
// the key is present. The two-value (comma-ok) form is required: a key whose
// value is the empty string must be distinguishable from an absent key,
// otherwise WAL replay (M4) and snapshot restore (M8) cannot be deterministic.
//
// Get is safe to call on a Store with a nil kv map (it will report ok=false
// for every key), but callers should still construct via New.
func (s *Store) Get(key string) (value string, ok bool) {
	value, ok = s.kv[key]
	return value, ok
}

// Put inserts or overwrites the value for key.
//
// Put is idempotent: Put(k, v) followed by Put(k, v) is observationally
// equivalent to a single Put(k, v). This is what makes Put safe to retry
// across network failures; Append (next step) does not have this property,
// which is why M2 will introduce client request IDs and a dedup table.
//
// Invariant (Safety): immediately after Put(k, v) returns, Get(k) == (v, true)
// — until the next Put(k, _) or Append(k, _) on the same key.
//
// Put panics if s was constructed by zero value rather than New (the map
// would be nil). This is intentional; callers must use New.
func (s *Store) Put(key, value string) {
	s.kv[key] = value
}

// Append concatenates value onto the existing value for key. If key is
// absent, Append creates it with value (Go's map zero value is the empty
// string, so s.kv[key]+value is correct in both cases without a branch).
//
// Append is NOT idempotent: Append(k, "x") applied twice leaves k with "xx",
// not "x". This is by design and is the motivating example for the client
// request IDs and server-side duplicate-detection table introduced in M2.
//
// Invariant (Safety): immediately after Append(k, v) returns,
// Get(k) == (oldValue + v, true), where oldValue is the value previously
// associated with k or "" if k was absent.
//
// Concurrency note: today the read-modify-write below is a single statement
// run by the sole caller, and is safe. In M3, concurrent gRPC handlers can
// race here — another goroutine could Put between this method's read and
// its write, silently losing that Put. The sync.RWMutex added in M3 closes
// that window. Raft preserves determinism across replicas in M7 because
// every node applies operations in the same total order.
func (s *Store) Append(key, value string) {
	s.kv[key] = s.kv[key] + value
}
