// Package store implements the in-memory key-value state machine used by the
// distributed KV store.
//
// The state machine is intentionally minimal: a single map[string]string and
// three operations (Get, Put, Append). It performs no I/O, no logging, and
// no concurrency control. By construction it is deterministic: given the same
// starting state and the same ordered sequence of operations, every replica
// reaches the same state and produces the same outputs. This is the
// replicated state machine model that Raft delivers operations into in
// Milestone 7.
//
// In Milestone 1 the Store is exercised directly by tests; in later
// milestones it sits behind a gRPC handler (M3) and then a Raft apply loop
// (M7). Neither layer requires changes to this package's API.
package store
