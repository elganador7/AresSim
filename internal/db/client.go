package db

import (
	"context"
	"fmt"
	"sync"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

// Client wraps a surrealdb.DB connection authenticated and scoped to the
// aressim namespace and engine database. It is safe for concurrent use.
//
// Obtain a Client via Connect or Manager.NewClient.
type Client struct {
	db  *surrealdb.DB
	cfg Config
	mu  sync.RWMutex
}

// Connect dials the SurrealDB instance described by cfg, authenticates as
// root, and selects the configured namespace and database. It is typically
// called immediately after Manager.Start returns.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	db, err := surrealdb.New(fmt.Sprintf("ws://127.0.0.1:%d/rpc", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("dial surrealdb: %w", err)
	}

	c := &Client{db: db, cfg: cfg}
	if err := c.authenticate(ctx); err != nil {
		_ = db.Close(ctx)
		return nil, err
	}
	return c, nil
}

// DB returns the underlying surrealdb.DB. Callers should prefer the typed
// helper functions in the repository layer over calling this directly.
func (c *Client) DB() *surrealdb.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db.Close(ctx)
}

// Reconnect closes the existing connection and dials a fresh one.
// Called by the adjudicator health-check goroutine if the connection drops.
func (c *Client) Reconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.db.Close(ctx)
	db, err := surrealdb.New(fmt.Sprintf("ws://127.0.0.1:%d/rpc", c.cfg.Port))
	if err != nil {
		return fmt.Errorf("reconnect dial: %w", err)
	}
	c.db = db
	return c.authenticate(ctx)
}

func (c *Client) authenticate(ctx context.Context) error {
	_, err := c.db.SignIn(ctx, surrealdb.Auth{
		Username: c.cfg.Username,
		Password: c.cfg.Password,
	})
	if err != nil {
		return fmt.Errorf("surrealdb signin: %w", err)
	}
	if err := c.db.Use(ctx, c.cfg.Namespace, c.cfg.Database); err != nil {
		return fmt.Errorf("surrealdb use %s/%s: %w", c.cfg.Namespace, c.cfg.Database, err)
	}
	return nil
}
