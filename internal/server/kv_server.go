package server

import (
	"context"
	"fmt"

	"github.com/imPranav14/Distributed-KV-Store/internal/wal"
	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
)

// KvServer is the gRPC server for the KV service.
//
// Invariants:
//   - (Safety) Every RPC is serialized into a single Store operation.
//   - (Safety) The underlying Store is assumed to be concurrency-safe.
//   - (Liveness) RPC handlers must always return a response or a clear error.
//
// Request IDs are accepted on Put/Append today to reserve the API shape for
// future dedup support, but they are not yet used by this Milestone 3 server.
type KvServer struct {
	kv.UnimplementedKvServiceServer
	store *wal.WALStore
}

// NewKvServer returns a server ready to handle KV RPCs.
func NewKvServer(store *wal.WALStore) *KvServer {
	if store == nil {
		panic("server requires a non-nil WALStore")
	}
	return &KvServer{store: store}
}

func (s *KvServer) Get(ctx context.Context, req *kv.GetRequest) (*kv.GetResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("GetRequest cannot be nil")
	}

	value, ok := s.store.Get(req.Key)
	status := kv.Status_STATUS_OK
	if !ok {
		status = kv.Status_STATUS_NOT_FOUND
	}

	return &kv.GetResponse{
		Status: status,
		Value:  value,
	}, nil
}

func (s *KvServer) Put(ctx context.Context, req *kv.PutRequest) (*kv.PutResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("PutRequest cannot be nil")
	}

	if err := s.store.Put(req.Key, req.Value); err != nil {
		return nil, fmt.Errorf("put failed: %w", err)
	}
	return &kv.PutResponse{Status: kv.Status_STATUS_OK}, nil
}

func (s *KvServer) Append(ctx context.Context, req *kv.AppendRequest) (*kv.AppendResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("AppendRequest cannot be nil")
	}

	if err := s.store.Append(req.Key, req.Value); err != nil {
		return nil, fmt.Errorf("append failed: %w", err)
	}
	return &kv.AppendResponse{Status: kv.Status_STATUS_OK}, nil
}
