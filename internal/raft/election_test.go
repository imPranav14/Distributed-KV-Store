package raft

import (
	"context"
	"testing"

	raftpb "github.com/imPranav14/Distributed-KV-Store/proto/raft"
)

func TestRequestVote_ServerGrantsVoteOnNewTerm(t *testing.T) {
	node := NewNode("node-1")
	srv := NewServer(node)

	resp, err := srv.RequestVote(context.Background(), &raftpb.RequestVoteRequest{
		Term:         1,
		CandidateId:  "candidate-1",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	if err != nil {
		t.Fatalf("RequestVote failed: %v", err)
	}
	if !resp.VoteGranted {
		t.Fatal("expected vote granted")
	}
	if resp.Term != 1 {
		t.Fatalf("expected term 1, got %d", resp.Term)
	}
}

func TestRequestVote_ServerRejectsOlderTerm(t *testing.T) {
	node := NewNode("node-1")
	node.CurrentTerm = 2
	srv := NewServer(node)

	resp, err := srv.RequestVote(context.Background(), &raftpb.RequestVoteRequest{
		Term:         1,
		CandidateId:  "candidate-1",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	if err != nil {
		t.Fatalf("RequestVote failed: %v", err)
	}
	if resp.VoteGranted {
		t.Fatal("expected vote denied for older term")
	}
	if resp.Term != 2 {
		t.Fatalf("expected current term 2, got %d", resp.Term)
	}
}

func TestRequestVote_ServerRejectsSecondVoteInTerm(t *testing.T) {
	node := NewNode("node-1")
	node.CurrentTerm = 1
	node.VotedFor = "candidate-1"
	srv := NewServer(node)

	resp, err := srv.RequestVote(context.Background(), &raftpb.RequestVoteRequest{
		Term:         1,
		CandidateId:  "candidate-2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	if err != nil {
		t.Fatalf("RequestVote failed: %v", err)
	}
	if resp.VoteGranted {
		t.Fatal("expected vote denied for second candidate in same term")
	}
	if resp.Term != 1 {
		t.Fatalf("expected term 1, got %d", resp.Term)
	}
}
