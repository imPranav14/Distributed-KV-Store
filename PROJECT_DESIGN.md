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

- Chose `string` values for `Put` and `Append` to match the current `internal/store` API and keep the first networking milestone simple.
- Chose a response `Status` enum instead of a separate `found` flag for `Get`, since it is more extensible and better aligned with RPC semantics.
- Planned client-side retry behavior around timeout/transient failures with stable request IDs, but deferred dedup enforcement to M7.

### Milestone 3 implementation details

- Added `internal/server/kv_server.go` to expose the generated `KvService` over gRPC.
- Server methods are thin wrappers around `internal/store`, reserving request ID fields for future dedup support.
- Added concurrency safety to `internal/store` with a `sync.RWMutex` so gRPC handlers can safely access shared state.
- Added `internal/client/client.go` with a `Client` type, request-ID generation, timeout-bound RPC attempts, and retry-on-retryable transport failures.
- Implemented `internal/server/kv_server_test.go` and `internal/client/client_test.go` to validate the end-to-end gRPC wiring using an in-memory bufconn transport.

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

- Project is now implementing Milestone 3 server and client wiring.
- Completed `proto/kv/kv.proto` and `proto/raft/raft.proto` design.
- Added `internal/server/kv_server.go` and `internal/client/client.go`.
- Added concurrency safety to `internal/store` with `sync.RWMutex` for gRPC handler safety.
- Added tests for both server and client implementation.
- Verified the implementation with `go test ./internal/...`.
- The selected API shape remains:
  - `Put` / `Append` values use `string`
  - `Get` returns a `Status` enum with `OK`, `NOT_FOUND`, and `ERROR`
  - request IDs are included in write requests so the client can safely retry on timeouts
- Next micro-step: add the first server binary entrypoint and internal/config layout for node startup.
