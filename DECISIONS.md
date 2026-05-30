# Design Decisions

Append-only log of design decisions, tradeoffs, and pedagogical choices.
New entries go at the bottom. Each entry follows:

> **D-NNN — Title** *(YYYY-MM-DD, status)*
> **Context.** What problem or question prompted this.
> **Decision.** What we chose.
> **Tradeoff.** What we accepted by choosing it.
> **Status.** `ACCEPTED` / `DEFERRED` / `SUPERSEDED-BY-DXXX`.

## Inherited scope (from `working.md`)

Crash-stop failure model only (no Byzantine). No TLS or authentication.
Single-region / single-DC. No automatic rebalancing. No external monitoring
stack. These framing constraints are not re-decided per entry below.

---

## Milestone 0 — bootstrap decisions

### D-001 — Go module path *(2026-05-22, ACCEPTED)*
**Context.** Need a canonical import path before any Go code is written.
**Decision.** `github.com/imPranav14/Distributed-KV-Store`, matching the GitHub remote.
**Tradeoff.** The local folder is `KV store` (with a space) and differs from
both the module path and the GitHub repo name. Go tooling is unaffected;
documented here for future readers.

### D-002 — KV API surface *(2026-05-22, ACCEPTED)*
**Context.** `working.md` proposed `Get` / `Put` / `Delete`. MIT 6.5840 Lab 3
uses `Get` / `Put` / `Append`.
**Decision.** Use `Get` / `Put` / `Append`. No `Delete` in v1.
**Tradeoff.** `Append` is non-idempotent on retry, which forces real
client-side request IDs and server-side duplicate detection — pedagogically
the point. `Delete` can be added later as a state-machine change only.

### D-003 — Value type *(2026-05-22, ACCEPTED)*
**Context.** State machine type for keys and values.
**Decision.** `string → string`, backed by `map[string]string`. Matches Lab 3.
**Tradeoff.** Cannot store arbitrary binary blobs. Widening to `[]byte` is
trivial *before* the WAL format is fixed in M4; harder after.

### D-004 — Consistency model *(2026-05-22, ACCEPTED)*
**Context.** What guarantees do clients see?
**Decision.** Strict linearizability. All operations (including `Get`) go
through Raft consensus, as in Lab 3A. No read-index, no lease reads.
**Tradeoff.** Every read costs one Raft round-trip. Throughput is bounded by
the leader's log append rate. Read-index and lease reads deferred to M10 as
an explicit optimization exercise.

### D-005 — Cluster membership *(2026-05-22, ACCEPTED)*
**Context.** Can the peer set change at runtime?
**Decision.** Static. Peers are fixed at process start via env/flags.
Matches Lab 2/3/4.
**Tradeoff.** No live add/remove of nodes. Joint consensus (Raft §6) is an
explicitly deprioritized extension.

### D-006 — Durability policy *(2026-05-22, ACCEPTED)*
**Context.** When must a write be on stable storage?
**Decision.** `fsync` before responding to RPCs, per Raft paper Figure 2 and
the semantic guarantee of MIT's `Persister` in Lab 2C.
**Tradeoff.** One disk sync per write. Group commit / batched fsync deferred
to M10 as a performance optimization.

### D-007 — Snapshot encoding *(2026-05-22, ACCEPTED)*
**Context.** How are state-machine snapshots serialized for `InstallSnapshot`
and on-disk storage?
**Decision.** Protobuf. The snapshot payload is opaque to Raft (`bytes`);
the KV state machine defines its own `KVSnapshot` proto message.
**Tradeoff.** Diverges from MIT 6.5840, which uses `labgob`. Accepted to keep
a single serialization framework across RPCs, WAL, and snapshots.

### D-008 — Local runtime modes *(2026-05-22, ACCEPTED)*
**Context.** How do we run multi-node clusters during development?
**Decision.** Three modes, in order of fidelity:
1. **In-process** goroutine clusters for unit/integration tests (most
   MIT-like; fastest).
2. **`make run-local`** — 3 OS processes on distinct ports, no Docker.
3. **`make up`** — Docker Compose, named volumes, simulates real deployment.

**Tradeoff.** Three configurations to keep aligned. Pays for itself during
Raft debugging.

### D-009 — Logging *(2026-05-22, ACCEPTED)*
**Context.** Structured logging from day one, without external dependencies.
**Decision.** `log/slog`. Default level `INFO`; `LOG_LEVEL=debug` overrides.
Standard fields from M5 onward: `node_id`, `term`, `role`, `commit_index`.
**Tradeoff.** Requires discipline to attach standard fields consistently;
will be enforced by a small logger helper introduced in M5.

### D-010 — Placeholder file style *(2026-05-22, ACCEPTED)*
**Context.** Empty directories don't survive Git. How do we mark intended
package boundaries before there is code in them?
**Decision.** We don't pre-stub. A package directory is created together
with its first real file *and* a `doc.go` that states the package's single
responsibility in one sentence.
**Tradeoff.** The module layout from `working.md` is not visible in git
history until each package lands. Acceptable.

### D-011 — File-creation philosophy *(2026-05-22, ACCEPTED)*
**Context.** Avoid premature scaffolding.
**Decision.** Files are created only when they are needed by the current
milestone. No `.gitkeep` files, no skeleton packages, no `docker-compose.yml`
before the binary exists.
**Tradeoff.** Each milestone begins with a small amount of layout work as
new directories appear for the first time.

---

## Open questions (revisit before the named milestone)

### D-012 — Sequencing of idempotency work *(2026-05-22, ACCEPTED-AS)*
`working.md` originally put client request IDs and server-side dedup (M2)
before the gRPC layer (M3). Idempotency cannot be meaningfully built or
tested without an RPC boundary, so we accept a revised sequencing:
- M3: gRPC + client + request IDs
- M7: Raft KV + dedup table
This preserves the pedagogical flow and avoids premature dedup-table design
before Raft is in place.
**Revisit:** none; this is the accepted plan.

### D-013 — WAL as standalone milestone vs. folded into Raft persistence *(2026-05-22, DEFERRED)*
`working.md` M4 introduces a single-node WAL, but in standard Raft the Raft
log *is* the persistent log; there is no separate WAL underneath.
**Options:** (a) keep M4 as an intentional throwaway exercise that teaches
`fsync`, replay, and recovery in isolation, then retire that code in M5;
(b) skip M4 and learn the same concepts inside M5/M6 Raft persistence.
**Revisit:** at the close of M3.
