package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/imPranav14/Distributed-KV-Store/internal/config"
	"github.com/imPranav14/Distributed-KV-Store/internal/raft"
	"github.com/imPranav14/Distributed-KV-Store/internal/server"
	"github.com/imPranav14/Distributed-KV-Store/internal/wal"
	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
	raftpb "github.com/imPranav14/Distributed-KV-Store/proto/raft"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	listener, grpcServer, walStore, err := startNode(cfg)
	if err != nil {
		log.Fatalf("start node: %v", err)
	}
	defer func() {
		grpcServer.GracefulStop()
		if err := walStore.Close(); err != nil {
			log.Printf("error closing WAL store: %v", err)
		}
	}()

	log.Printf("node %s listening on %s, wal path %s", cfg.NodeID, listener.Addr(), cfg.WALPath())

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	<-stopCh
	log.Println("shutting down")
}

func startNode(cfg *config.Config) (net.Listener, *grpc.Server, *wal.WALStore, error) {
	walStore, err := wal.NewStoreWithWAL(cfg.WALPath())
	if err != nil {
		return nil, nil, nil, err
	}

	grpcServer := grpc.NewServer()
	kv.RegisterKvServiceServer(grpcServer, server.NewKvServer(walStore))

	raftNode := raft.NewNode(cfg.NodeID)
	raftpb.RegisterRaftServiceServer(grpcServer, raft.NewServer(raftNode))

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		walStore.Close()
		return nil, nil, nil, err
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			log.Printf("serve grpc: %v", err)
		}
	}()

	return listener, grpcServer, walStore, nil
}
