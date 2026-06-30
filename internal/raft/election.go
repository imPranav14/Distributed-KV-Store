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
