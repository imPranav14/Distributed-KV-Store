# Distributed KV Store

A fault-tolerant, linearizable distributed key-value store built from scratch in
Go, following the pedagogy of MIT 6.5840 (Distributed Systems). gRPC for
transport, Raft for consensus, no third-party KV or consensus libraries.

**Status:** Milestone 4 completed; Milestone 5 Raft election scaffolding started.

## Documents

- [`working.md`](./working.md) — original project brief and roadmap.
- [`DECISIONS.md`](./DECISIONS.md) — running log of design decisions and tradeoffs.
- [`PROJECT_DESIGN.md`](./PROJECT_DESIGN.md) — chronological design history, tradeoffs, and plan changes.

## Quick start

```sh
make help       # list available targets
make test       # run tests
make vet        # static analysis
```

The project currently has a working WAL-backed KV server and a minimal Raft node scaffold for leader-election state transitions.

### What is implemented
- gRPC KV service over protobuf
- WAL persistence and replay for durable writes
- a small `internal/raft` package with node role transitions and election timeout logic

### How it is implemented
- The KV server uses a WAL-backed store so writes are appended before the in-memory state is updated.
- The Raft package models leader election state explicitly with a small `Node` struct and simple transition methods.

Most targets are placeholders until the milestone that wires them up; see
`make help` for the per-target milestone tag.

## Roadmap

See `working.md` §Milestone Roadmap. Briefly:

M0 architecture → M1 single-node KV → M3 gRPC + client + request IDs → M4 WAL →
M5/M6 Raft → M7 KV-on-Raft + dedup table → M8 snapshots → M9 sharding → M10 hardening.
