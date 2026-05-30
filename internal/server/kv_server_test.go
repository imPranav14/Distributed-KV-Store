package server

import (
	"context"
	"testing"

	"github.com/imPranav14/Distributed-KV-Store/internal/store"
	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
)

func TestKvServer_GetPutAppend(t *testing.T) {
	s := store.New()
	srv := NewKvServer(s)
	ctx := context.Background()

	putResp, err := srv.Put(ctx, &kv.PutRequest{Key: "k", Value: "v", RequestId: "id-1"})
	if err != nil {
		t.Fatalf("Put returned unexpected error: %v", err)
	}
	if putResp.Status != kv.Status_STATUS_OK {
		t.Fatalf("Put returned status %v, want STATUS_OK", putResp.Status)
	}

	getResp, err := srv.Get(ctx, &kv.GetRequest{Key: "k"})
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if getResp.Status != kv.Status_STATUS_OK {
		t.Fatalf("Get returned status %v, want STATUS_OK", getResp.Status)
	}
	if getResp.Value != "v" {
		t.Fatalf("Get returned value %q, want %q", getResp.Value, "v")
	}

	appendResp, err := srv.Append(ctx, &kv.AppendRequest{Key: "k", Value: "x", RequestId: "id-2"})
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if appendResp.Status != kv.Status_STATUS_OK {
		t.Fatalf("Append returned status %v, want STATUS_OK", appendResp.Status)
	}

	getResp, err = srv.Get(ctx, &kv.GetRequest{Key: "k"})
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if getResp.Status != kv.Status_STATUS_OK {
		t.Fatalf("Get returned status %v, want STATUS_OK", getResp.Status)
	}
	if getResp.Value != "vx" {
		t.Fatalf("Get returned value %q, want %q", getResp.Value, "vx")
	}
}
