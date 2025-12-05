package embedding

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "llm-router/services/router/pkg/pb"
)

type Client struct {
	conn    *grpc.ClientConn
	client  pb.EmbeddingServiceClient
	timeout time.Duration
}

func NewClient(address string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to embedding service: %w", err)
	}

	return &Client{
		conn:    conn,
		client:  pb.NewEmbeddingServiceClient(conn),
		timeout: timeout,
	}, nil
}

type EmbedResult struct {
	Vector    []float32
	LatencyMs float32
}

func (c *Client) Embed(ctx context.Context, text string) (*EmbedResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.Embed(ctx, &pb.EmbedRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}

	return &EmbedResult{
		Vector:    resp.Vector,
		LatencyMs: resp.LatencyMs,
	}, nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.client.Embed(ctx, &pb.EmbedRequest{Text: "health"})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}
