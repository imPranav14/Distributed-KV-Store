package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	kv "github.com/imPranav14/Distributed-KV-Store/proto/kv"
)

// Config controls the behavior of the KV client.
type Config struct {
	Addr         string
	ClientID     string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	Dialer       func(context.Context, string) (net.Conn, error)
}

// Client is a gRPC client for the KV service.
//
// Invariants:
//   - (Safety) request IDs are stable for the duration of a single Put/Append
//     invocation so retries reuse the same identifier.
//   - (Liveness) each RPC attempt is bounded by Timeout and the caller's ctx.
//   - (Safety) only retryable transport failures are retried.
type Client struct {
	cfg      Config
	conn     *grpc.ClientConn
	kvClient kv.KvServiceClient
	seq      uint64
}

// NewClient creates a new KV client and opens a gRPC connection.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("client address is required")
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "client"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
	}

	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if cfg.Dialer != nil {
		dialOptions = append(dialOptions, grpc.WithContextDialer(cfg.Dialer))
	}

	conn, err := grpc.DialContext(ctx, cfg.Addr, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("dial kv server: %w", err)
	}

	return &Client{
		cfg:      cfg,
		conn:     conn,
		kvClient: kv.NewKvServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) nextRequestID() string {
	seq := atomic.AddUint64(&c.seq, 1)
	return fmt.Sprintf("%s-%d", c.cfg.ClientID, seq)
}

func (c *Client) doWithRetry(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
		err := op(callCtx)
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err

		if !isRetryable(err) || attempt == c.cfg.MaxRetries-1 {
			break
		}

		select {
		case <-time.After(c.cfg.RetryBackoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

// Get fetches a value by key. A NOT_FOUND response is returned as a valid
// response, not an error.
func (c *Client) Get(ctx context.Context, key string) (*kv.GetResponse, error) {
	var resp *kv.GetResponse
	err := c.doWithRetry(ctx, func(callCtx context.Context) error {
		result, err := c.kvClient.Get(callCtx, &kv.GetRequest{Key: key})
		if err != nil {
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == kv.Status_STATUS_ERROR {
		return resp, fmt.Errorf("server error: %s", resp.ErrorMessage)
	}
	return resp, nil
}

// Put stores a value with a stable request ID so retries reuse the same ID.
func (c *Client) Put(ctx context.Context, key, value string) (*kv.PutResponse, error) {
	req := &kv.PutRequest{
		Key:       key,
		Value:     value,
		RequestId: c.nextRequestID(),
	}
	var resp *kv.PutResponse
	err := c.doWithRetry(ctx, func(callCtx context.Context) error {
		result, err := c.kvClient.Put(callCtx, req)
		if err != nil {
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == kv.Status_STATUS_ERROR {
		return resp, fmt.Errorf("server error: %s", resp.ErrorMessage)
	}
	return resp, nil
}

// Append appends to an existing value using a request ID. Retries may be
// unsafe until dedup support is added on the server side.
func (c *Client) Append(ctx context.Context, key, value string) (*kv.AppendResponse, error) {
	req := &kv.AppendRequest{
		Key:       key,
		Value:     value,
		RequestId: c.nextRequestID(),
	}
	var resp *kv.AppendResponse
	err := c.doWithRetry(ctx, func(callCtx context.Context) error {
		result, err := c.kvClient.Append(callCtx, req)
		if err != nil {
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == kv.Status_STATUS_ERROR {
		return resp, fmt.Errorf("server error: %s", resp.ErrorMessage)
	}
	return resp, nil
}
