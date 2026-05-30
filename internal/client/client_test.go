package client

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/imPranav14/Distributed-KV-Store/internal/server"
	"github.com/imPranav14/Distributed-KV-Store/internal/store"
	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
)

const bufSize = 1024 * 1024

func dialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.Dial()
	}
}

func TestClient_GetPutAppend(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	store := store.New()
	kvServer := server.NewKvServer(store)
	kv.RegisterKvServiceServer(s, kvServer)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	ctx := context.Background()
	client, err := NewClient(ctx, Config{
		Addr:         "bufnet",
		ClientID:     "test-client",
		Timeout:      1 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 10 * time.Millisecond,
		Dialer:       dialer(lis),
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if _, err := client.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	getResp, err := client.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if getResp.Status != kv.Status_STATUS_OK || getResp.Value != "v" {
		t.Fatalf("Get returned %+v, want STATUS_OK and value v", getResp)
	}

	if _, err := client.Append(ctx, "k", "x"); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	getResp, err = client.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if getResp.Status != kv.Status_STATUS_OK || getResp.Value != "vx" {
		t.Fatalf("Get returned %+v, want STATUS_OK and value vx", getResp)
	}
}
