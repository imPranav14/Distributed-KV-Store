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
