package raft

import (
	"context"
	"fmt"

	raftpb "github.com/imPranav14/Distributed-KV-Store/proto/raft"
	"google.golang.org/grpc"
)

// Client is a lightweight Raft RPC client for election requests.
type Client struct {
	client raftpb.RaftServiceClient
}

// NewClient creates a Raft RPC client over the given gRPC connection.
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{client: raftpb.NewRaftServiceClient(conn)}
}

// RequestVote sends a RequestVote RPC to the peer.
func (c *Client) RequestVote(ctx context.Context, term int64, candidateID string, lastLogIndex, lastLogTerm int64) (*raftpb.RequestVoteResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("raft client is not initialized")
	}

	return c.client.RequestVote(ctx, &raftpb.RequestVoteRequest{
		Term:         term,
		CandidateId:  candidateID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	})
}

// AppendEntries sends an AppendEntries (heartbeat/replication) RPC to the peer.
func (c *Client) AppendEntries(ctx context.Context, term int64, leaderID string, prevLogIndex, prevLogTerm int64, entries []*raftpb.LogEntry, leaderCommit int64) (*raftpb.AppendEntriesResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("raft client is not initialized")
	}

	return c.client.AppendEntries(ctx, &raftpb.AppendEntriesRequest{
		Term:         term,
		LeaderId:     leaderID,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: leaderCommit,
	})
}
