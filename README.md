# Distributed KV Store

A fault-tolerant, linearizable distributed key-value store built from scratch in
Go, following the pedagogy of MIT 6.5840 (Distributed Systems). gRPC for
transport, Raft for consensus, no third-party KV or consensus libraries.

**Status:** Milestone 0 — Requirements and Architecture.

## Documents

- [`working.md`](./working.md) — original project brief and roadmap.
- [`DECISIONS.md`](./DECISIONS.md) — running log of design decisions and tradeoffs.

## Quick start

```sh
make help       # list available targets
make test       # run tests (none yet at M0)
make vet        # static analysis
```

Most targets are placeholders until the milestone that wires them up; see
`make help` for the per-target milestone tag.

## Roadmap

See `working.md` §Milestone Roadmap. Briefly:

M0 architecture → M1 single-node KV → M2/M3 client + gRPC → M4 WAL →
M5/M6 Raft → M7 KV-on-Raft → M8 snapshots → M9 sharding → M10 hardening.
