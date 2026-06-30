package raft

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Role represents the current Raft node role.
type Role string

const (
	RoleFollower  Role = "follower"
	RoleCandidate Role = "candidate"
	RoleLeader    Role = "leader"
)

// Node is the minimal Raft node state for Milestone 5.
//
// It intentionally keeps only the pieces needed for leader election:
//   - current term
//   - voted-for state
//   - current role
//   - election timeout
//
// This is a small foundation that can later grow into RPC handling and
// replication logic without changing its public shape unnecessarily.
type Node struct {
	mu sync.Mutex

	ID          string
	Role        Role
	CurrentTerm int64
	VotedFor    string

	ElectionTimeout time.Duration
	lastHeartbeat   time.Time

	// Peers holds gRPC clients to other Raft peers. It may be nil.
	Peers []*Client
}

// NewNode creates a follower node with default election timing.
func NewNode(id string) *Node {
	if id == "" {
		panic("raft node id cannot be empty")
	}
	return &Node{
		ID:              id,
		Role:            RoleFollower,
		ElectionTimeout: 150 * time.Millisecond,
		lastHeartbeat:   time.Now(),
	}
}

// RoleString returns the current role as a string.
func (n *Node) RoleString() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return string(n.Role)
}

// StartElection begins a new election round.
func (n *Node) StartElection() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.CurrentTerm++
	n.VotedFor = n.ID
	n.Role = RoleCandidate
	n.lastHeartbeat = time.Now()

	return nil
}

// BecomeLeader marks the node as leader for the current term.
func (n *Node) BecomeLeader() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Role = RoleLeader
	n.lastHeartbeat = time.Now()
	return nil
}

// BecomeFollower resets the node to follower state and records the term.
func (n *Node) BecomeFollower(term int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term < n.CurrentTerm {
		return fmt.Errorf("term %d is older than current term %d", term, n.CurrentTerm)
	}
	if term > n.CurrentTerm {
		n.CurrentTerm = term
		n.VotedFor = ""
	}
	if n.Role == RoleLeader && term > n.CurrentTerm {
		n.Role = RoleFollower
	}
	n.Role = RoleFollower
	n.lastHeartbeat = time.Now()
	return nil
}

// ResetElectionTimer updates the heartbeat timestamp.
func (n *Node) ResetElectionTimer() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastHeartbeat = time.Now()
}

// ElectionExpired reports whether the election timeout has elapsed.
func (n *Node) ElectionExpired(now time.Time) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return now.Sub(n.lastHeartbeat) >= n.ElectionTimeout
}

// CurrentState returns a simple snapshot of the node state.
type StateSnapshot struct {
	ID              string
	Role            Role
	CurrentTerm     int64
	VotedFor        string
	ElectionTimeout time.Duration
}

// Snapshot returns a read-only snapshot for debugging or tests.
func (n *Node) Snapshot() StateSnapshot {
	n.mu.Lock()
	defer n.mu.Unlock()
	return StateSnapshot{
		ID:              n.ID,
		Role:            n.Role,
		CurrentTerm:     n.CurrentTerm,
		VotedFor:        n.VotedFor,
		ElectionTimeout: n.ElectionTimeout,
	}
}

// SetPeers registers the peer clients for this node. It overwrites any
// existing peer list.
func (n *Node) SetPeers(peers []*Client) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Peers = peers
}

// RunElectionLoop runs a simple election loop until the provided context
// is cancelled. When the election timeout expires the node starts an
// election, sends `RequestVote` RPCs to peers and becomes leader if it
// gathers a majority of votes.
func (n *Node) RunElectionLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if !n.ElectionExpired(now) {
				continue
			}

			// Start a new election
			if err := n.StartElection(); err != nil {
				// shouldn't happen but skip this round
				continue
			}

			// snapshot peers to avoid holding locks while dialing
			n.mu.Lock()
			peers := make([]*Client, 0, len(n.Peers))
			peers = append(peers, n.Peers...)
			term := n.CurrentTerm
			n.mu.Unlock()

			// count self vote
			votes := 1

			// context for RPCs
			rpcCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			var wg sync.WaitGroup
			results := make(chan bool, len(peers))

			for _, p := range peers {
				if p == nil {
					continue
				}
				wg.Add(1)
				go func(c *Client) {
					defer wg.Done()
					resp, err := c.RequestVote(rpcCtx, term, n.ID, 0, 0)
					if err != nil || resp == nil {
						return
					}
					if resp.Term > term {
						// discovered higher term; step down
						_ = n.BecomeFollower(resp.Term)
						return
					}
					if resp.VoteGranted {
						results <- true
					}
				}(p)
			}

			wg.Wait()
			cancel()
			close(results)

			for r := range results {
				if r {
					votes++
				}
			}

			// majority check (peers + self)
			total := len(peers) + 1
			if votes*2 > total {
				_ = n.BecomeLeader()
			}
		}
	}
}
