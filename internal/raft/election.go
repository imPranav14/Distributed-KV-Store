package raft

import (
	"context"
	"fmt"
	"time"

	raftpb "github.com/imPranav14/Distributed-KV-Store/proto/raft"
)

// Server implements the RaftService RPC interface for election requests.
// It currently only supports RequestVote for Milestone 5.
type Server struct {
	raftpb.UnimplementedRaftServiceServer
	node *Node
}

// NewServer returns a Raft RPC server backed by a Raft node.
func NewServer(node *Node) *Server {
	if node == nil {
		panic("raft server requires a non-nil node")
	}
	return &Server{node: node}
}

// RequestVote handles a candidate's election request.
//
// This implementation enforces Raft safety for terms and vote assignment,
// but does not yet compare log freshness. That check will be added once the
// log layer exists.
func (s *Server) RequestVote(ctx context.Context, req *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("RequestVoteRequest cannot be nil")
	}

	term, voteGranted := s.node.HandleRequestVote(req.Term, req.CandidateId)
	return &raftpb.RequestVoteResponse{Term: term, VoteGranted: voteGranted}, nil
}

// AppendEntries handles leader heartbeats and log replication requests.
// For Milestone 5 this implementation accepts empty heartbeats and
// resets the follower's election timer when the incoming term is
// at least the node's current term.
func (s *Server) AppendEntries(ctx context.Context, req *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("AppendEntriesRequest cannot be nil")
	}

	// If term is older, reject but return current term
	n := s.node
	n.mu.Lock()
	current := n.CurrentTerm
	n.mu.Unlock()

	if req.Term < current {
		return &raftpb.AppendEntriesResponse{Term: current, Success: false}, nil
	}

	// If request term is newer, step down and update term
	if req.Term > current {
		_ = n.BecomeFollower(req.Term)
	}

	// Reset election timer on valid heartbeat
	n.ResetElectionTimer()

	return &raftpb.AppendEntriesResponse{Term: n.CurrentTerm, Success: true}, nil
}

// HandleRequestVote decides whether the local node can grant a vote.
// It updates term and vote state atomically.
func (n *Node) HandleRequestVote(term int64, candidateID string) (int64, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term < n.CurrentTerm {
		return n.CurrentTerm, false
	}

	if term > n.CurrentTerm {
		n.CurrentTerm = term
		n.VotedFor = ""
		n.Role = RoleFollower
	}

	if n.VotedFor == "" || n.VotedFor == candidateID {
		n.VotedFor = candidateID
		n.Role = RoleFollower
		n.lastHeartbeat = time.Now()
		return n.CurrentTerm, true
	}

	return n.CurrentTerm, false
}
