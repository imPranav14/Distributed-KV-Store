package main

import (
	"context"
	"testing"
	"time"

	"github.com/imPranav14/Distributed-KV-Store/internal/client"
	"github.com/imPranav14/Distributed-KV-Store/internal/config"
	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
)

func TestNodeStartup_WALRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		NodeID:   "test-node",
		WALDir:   tmpDir,
		GRPCAddr: "127.0.0.1:0",
	}

	listener, grpcServer, walStore, err := startNode(cfg)
	if err != nil {
		t.Fatalf("startNode failed: %v", err)
	}
	defer func() {
		grpcServer.GracefulStop()
		_ = walStore.Close()
	}()

	addr := listener.Addr().String()
	clientA, err := client.NewClient(context.Background(), client.Config{
		Addr:       addr,
		ClientID:   "smoke-client",
		Timeout:    1 * time.Second,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer clientA.Close()

	if _, err := clientA.Put(context.Background(), "k", "v"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	grpcServer.GracefulStop()
	if err := walStore.Close(); err != nil {
		t.Fatalf("close wal store failed: %v", err)
	}

	// Restart the node to verify WAL replay.
	listener2, grpcServer2, walStore2, err := startNode(cfg)
	if err != nil {
		t.Fatalf("restart node failed: %v", err)
	}
	defer func() {
		grpcServer2.GracefulStop()
		_ = walStore2.Close()
	}()

	clientB, err := client.NewClient(context.Background(), client.Config{
		Addr:       listener2.Addr().String(),
		ClientID:   "smoke-client-2",
		Timeout:    1 * time.Second,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer clientB.Close()

	getResp, err := clientB.Get(context.Background(), "k")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if getResp.Status != kv.Status_STATUS_OK {
		t.Fatalf("expected status STATUS_OK, got %v", getResp.Status)
	}
	if getResp.Value != "v" {
		t.Fatalf("expected value v after restart, got %q", getResp.Value)
	}
}
