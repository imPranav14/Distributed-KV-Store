package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultNodeID   = "node-1"
	defaultWALDir   = "data"
	defaultWALFile  = "wal.log"
	defaultGRPCAddr = "127.0.0.1:50051"
)

type Config struct {
	NodeID   string
	WALDir   string
	GRPCAddr string
}

func Parse() (*Config, error) {
	return ParseArgs(os.Args[1:])
}

func ParseArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet("node", flag.ContinueOnError)
	nodeID := fs.String("node-id", firstNonEmpty(os.Getenv("NODE_ID"), defaultNodeID), "unique node identifier")
	walDir := fs.String("wal-dir", firstNonEmpty(os.Getenv("WAL_DIR"), defaultWALDir), "directory for WAL files")
	grpcAddr := fs.String("grpc-addr", firstNonEmpty(os.Getenv("GRPC_ADDR"), defaultGRPCAddr), "gRPC listen address")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *nodeID == "" {
		return nil, fmt.Errorf("node-id cannot be empty")
	}

	return &Config{
		NodeID:   *nodeID,
		WALDir:   *walDir,
		GRPCAddr: *grpcAddr,
	}, nil
}

func (c *Config) WALPath() string {
	return filepath.Join(c.WALDir, c.NodeID+"-"+defaultWALFile)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
