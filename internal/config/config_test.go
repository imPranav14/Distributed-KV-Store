package config

import (
	"path/filepath"
	"testing"
)

func TestParseArgs_Defaults(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}

	if cfg.NodeID != defaultNodeID {
		t.Fatalf("expected node ID %q, got %q", defaultNodeID, cfg.NodeID)
	}
	if cfg.WALDir != defaultWALDir {
		t.Fatalf("expected WAL dir %q, got %q", defaultWALDir, cfg.WALDir)
	}
	if cfg.GRPCAddr != defaultGRPCAddr {
		t.Fatalf("expected gRPC addr %q, got %q", defaultGRPCAddr, cfg.GRPCAddr)
	}
	if got := cfg.WALPath(); got != filepath.Join(defaultWALDir, defaultNodeID+"-"+defaultWALFile) {
		t.Fatalf("expected WAL path %q, got %q", filepath.Join(defaultWALDir, defaultNodeID+"-"+defaultWALFile), got)
	}
}

func TestParseArgs_CustomFlags(t *testing.T) {
	cfg, err := ParseArgs([]string{"--node-id", "alpha", "--wal-dir", "data/alpha", "--grpc-addr", "127.0.0.1:6000"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}

	if cfg.NodeID != "alpha" {
		t.Fatalf("expected node ID alpha, got %q", cfg.NodeID)
	}
	if cfg.WALDir != "data/alpha" {
		t.Fatalf("expected WAL dir data/alpha, got %q", cfg.WALDir)
	}
	if cfg.GRPCAddr != "127.0.0.1:6000" {
		t.Fatalf("expected gRPC addr 127.0.0.1:6000, got %q", cfg.GRPCAddr)
	}
}
