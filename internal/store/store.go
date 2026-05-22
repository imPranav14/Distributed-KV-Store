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
