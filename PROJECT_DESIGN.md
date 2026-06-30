# Project Design History

This file captures the key design choices, tradeoffs, challenges, and plan changes made while building the distributed KV store.

## Purpose

- Keep a chronological record of architectural decisions.
- Explain why we chose one path over another.
- Document when and why the project plan changed.
- Preserve lessons learned as the system evolves.

## Milestones and Design Decisions

### Milestone 0 — Requirements and Architecture

- Defined the project as a fault-tolerant, linearizable distributed key-value store in Go.
- Chose gRPC and protobuf for all network communication.
- Agreed to use OS file I/O for persistence and no third-party consensus or KV libraries.
- Decided not to create files until they were needed by the current milestone, avoiding premature scaffolding.

### Milestone 1 — Single-Node KV Store

- Built `internal/store` as a pure in-memory state machine with `Get`, `Put`, and `Append`.
- Kept the API minimal and deterministic so it can later sit behind gRPC or Raft without API changes.
- Deliberately avoided concurrency control and persistence for this milestone.
- Tradeoff: simple and easy to test now, but not yet deployable as a real service.

### Design change: idempotency sequencing

- Original plan placed client request IDs and server-side dedup in Milestone 2, before gRPC.
- Challenge: idempotency cannot be meaningfully built or tested without an RPC boundary.
- Decision D-012 was revised from `DEFERRED` to `ACCEPTED-AS`.
- New plan:
  - M3: gRPC + client + request IDs
  - M7: Raft KV + dedup table
- Rationale: this avoids premature dedup-table architecture before Raft and preserves the pedagogical flow.

### Open question: WAL sequencing

- D-013 remains deferred.
- The question is whether to keep Milestone 4 as a standalone WAL exercise or fold persistence into Raft proper.
- This question will be revisited after Milestone 3.

### Milestone 3 API design decisions

- `Put` / `Append` value type:
  - Option A: use `string` values.
  - Option B: use `bytes` values.
  - Chosen: `string`.
  - Why: this matches the existing `internal/store` API, keeps the first networking milestone easy to reason about, and avoids introducing binary-handling complexity before the persistence and Raft layers are in place.

- `Get` response shape:
  - Option A: separate `found` boolean flag.
  - Option B: `Status` enum with `OK`, `NOT_FOUND`, `ERROR`.
  - Chosen: `Status` enum.
  - Why: an enum is more explicit for RPC semantics, makes the protocol more extensible, and separates application-level not-found semantics from transport-level failures.

- Retry semantics and request IDs:
  - Option A: no retries until server-side dedup exists.
  - Option B: allow retries on transport/timeouts using stable client-generated request IDs, but do not enforce dedup on server yet.
  - Chosen: Option B.
  - Why: this preserves the API shape needed for future dedup, enables realistic client behavior, and keeps the Milestone 3 server simple while leaving actual dedup enforcement for M7.

### Milestone 3 implementation details

- Added `internal/server/kv_server.go` to expose the generated `KvService` over gRPC.
- Server methods are thin wrappers around `internal/store`, reserving request ID fields for future dedup support.
- Added concurrency safety to `internal/store` with a `sync.RWMutex` so gRPC handlers can safely access shared state.
- Added `internal/client/client.go` with a `Client` type, request-ID generation, timeout-bound RPC attempts, and retry-on-retryable transport failures.
- Implemented `internal/server/kv_server_test.go` and `internal/client/client_test.go` to validate the end-to-end gRPC wiring using an in-memory bufconn transport.

### Milestone 4 WAL design decisions

- WAL serialization format:
  - Option A: use a protobuf schema in `proto/wal/wal.proto`.
  - Option B: use a custom binary or text format in internal code only.
  - Chosen: `proto/wal/wal.proto`.
  - Why: keeping WAL entries in protobuf matches the rest of the project, makes the on-disk schema explicit, and makes it easier to evolve the log format later.

- WAL entry fields:
  - Option A: persist only operation type, key, and value.
  - Option B: include `client_id` + numeric `request_id` for later dedup.
  - Chosen: include both `client_id` and `request_id`.
  - Why: request dedup is naturally keyed by client/request pair, and splitting the fields now avoids a later schema migration.

- Replay crash recovery semantics:
  - Option A: fail on any partial entry.
  - Option B: tolerate a truncated final entry and replay only complete records.
  - Chosen: tolerate truncated final entry.
  - Why: this is the standard append-only log recovery model, and it makes crash recovery robust without requiring extra end-of-log markers.

- WAL safety guard:
  - Option A: trust the length prefix and allocate each record blindly.
  - Option B: enforce a maximum reasonable record size before allocating.
  - Chosen: enforce a `MaxRecordSize` guard in replay.
  - Why: prevents corrupted or malicious WAL tails from causing unbounded memory allocation during recovery.

- Implementation approach:
  - Append each entry with an 8-byte length prefix.
  - Flush and `Sync()` the file on every write.
  - Replay by reading entry lengths and payloads, stopping cleanly on a truncated tail.
  - Encapsulate durability in `WAL` and state-machine application in `WALStore`.

## Tradeoffs and Challenges

- `store` package purity vs. early integration:
  - Chose to keep `internal/store` as a pure state machine without I/O or concurrency.
  - Benefit: easier reasoning, deterministic behavior, and later reuse under gRPC/Raft.

- Premature architecture vs. incremental progress:
  - Decision to avoid creating files/packages until needed kept the workspace small and focused.
  - Tradeoff: some layout work is deferred until later milestones, but the code remains cleaner.

- Idempotency timing:
  - Originally planned before gRPC, but the RPC boundary is required to implement and test request IDs properly.
  - This change prevents building a dedup layer that would be reworked once Raft arrives.

## Current status

- Project is now implementing Milestone 4 WAL persistence.
- Added `proto/wal/wal.proto` and generated `proto/wal/wal.pb.go` via `make proto`.
- Added `internal/wal/entry.go`, `internal/wal/wal.go`, and `internal/wal/wal_test.go`.
- Added a `WALStore` wrapper that appends to WAL before applying `Put`/`Append` to `internal/store`.
- Added `internal/config/config.go` for parsing node startup flags and deriving the WAL path.
- Added `wal.NewStoreWithWAL(path)` to open, replay, and return a WAL-backed state machine.
- Wired `internal/server/kv_server.go` to use `*wal.WALStore` so RPC writes are durably logged before applying in-memory state.
- Added `cmd/node/main.go` as a minimal node entrypoint that uses config, starts the WAL-backed KV server, and listens for gRPC.
- Added `cmd/node/main_test.go` to smoke-test node startup and WAL replay across restart.
- Added `run-local` to `Makefile` so `make run-local` starts the node entrypoint.
- Verified the implementation with `go test ./...`.
- The selected API shape remains:
  - `Put` / `Append` values use `string`
  - `Get` returns a `Status` enum with `OK`, `NOT_FOUND`, and `ERROR`
  - request IDs are included in write requests so the client can safely retry on timeouts
- Next micro-step: add a local run target and finalize node config/flags for repeated multi-node startup.

## Project log

- 2026-05-22: decided to revise idempotency sequencing. Chose M3 for request IDs and M7 for dedup tables so we do not prematurely design dedup before Raft.
- 2026-05-30: completed Milestone 3 gRPC/client design, generated proto stubs, and added server/client packages with in-memory bufconn tests.
- 2026-06-03: completed Milestone 4 WAL design and implementation. Added `proto/wal/wal.proto`, durable append-only logging, crash-tolerant replay, and integration tests.
- 2026-06-08: refined WAL schema to store `client_id` + numeric `request_id` separately and added `MaxRecordSize` validation to prevent oversized/corrupted replay allocations.
- 2026-06-09: added `internal/config/config.go`, wired `WALStore` into `internal/server/kv_server.go`, added `wal.NewStoreWithWAL(path)`, created `cmd/node/main.go`, added `cmd/node/main_test.go` for smoke testing restart recovery, added `run-local` to `Makefile`, and updated server/client tests to exercise WAL-backed writes.
- 2026-06-30: added Raft election scaffolding and started the first RPC-layer for elections.
  - Created `internal/raft/node.go` to encode Raft node roles (`follower`, `candidate`, `leader`) and election state.
  - Implemented `NewNode(id)` with a default election timeout and an internal heartbeat timestamp.
  - Added explicit transition methods: `StartElection()`, `BecomeLeader()`, `BecomeFollower(term)`, and `ResetElectionTimer()`.
  - Included `ElectionExpired(now)` so election timeout checks are deterministic and testable.
  - Added `StateSnapshot` and `Snapshot()` for safe read-only inspection in tests and future diagnostics.
  - Created `internal/raft/election.go` with a minimal `RequestVote` RPC implementation that enforces term safety and single-vote-per-term behavior.
  - Created `internal/raft/election_test.go` covering granting votes for a newer term, rejecting older-term requests, and denying second votes in the same term.
  - Updated `README.md` to document that Milestone 4 is complete and Milestone 5 Raft election scaffolding has begun.
