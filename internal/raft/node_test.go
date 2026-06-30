package raft

import (
	"testing"
	"time"
)

func TestNodeElectionStateTransitions(t *testing.T) {
	node := NewNode("node-1")

	if got := node.RoleString(); got != string(RoleFollower) {
		t.Fatalf("expected initial role %q, got %q", RoleFollower, got)
	}

	if err := node.StartElection(); err != nil {
		t.Fatalf("StartElection failed: %v", err)
	}
	if got := node.RoleString(); got != string(RoleCandidate) {
		t.Fatalf("expected role %q after election start, got %q", RoleCandidate, got)
	}
	if got := node.Snapshot().CurrentTerm; got != 1 {
		t.Fatalf("expected term 1 after election start, got %d", got)
	}
	if got := node.Snapshot().VotedFor; got != "node-1" {
		t.Fatalf("expected vote for self, got %q", got)
	}

	if err := node.BecomeLeader(); err != nil {
		t.Fatalf("BecomeLeader failed: %v", err)
	}
	if got := node.RoleString(); got != string(RoleLeader) {
		t.Fatalf("expected role %q after leader promotion, got %q", RoleLeader, got)
	}

	if err := node.BecomeFollower(2); err != nil {
		t.Fatalf("BecomeFollower failed: %v", err)
	}
	if got := node.Snapshot().CurrentTerm; got != 2 {
		t.Fatalf("expected term 2 after follower transition, got %d", got)
	}
}

func TestNodeElectionTimeout(t *testing.T) {
	node := NewNode("node-2")
	node.ResetElectionTimer()

	if node.ElectionExpired(time.Now().Add(-10 * time.Millisecond)) {
		t.Fatal("expected election to be fresh")
	}

	if !node.ElectionExpired(time.Now().Add(300 * time.Millisecond)) {
		t.Fatal("expected election timeout to expire")
	}
}
