package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

func NewClient(options *redis.Options) *Client {
	return &Client{Client: redis.NewClient(options)}
}

func (c *Client) PingContext(ctx context.Context) error {
	return c.Ping(ctx).Err()
}
