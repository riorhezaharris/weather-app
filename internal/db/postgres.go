package db

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReplicaPool distributes read queries across multiple PostgreSQL replicas
// using round-robin selection.
type ReplicaPool struct {
	pools   []*pgxpool.Pool
	counter atomic.Uint64
}

func NewReplicaPool(ctx context.Context, urls ...string) (*ReplicaPool, error) {
	pools := make([]*pgxpool.Pool, 0, len(urls))
	for _, url := range urls {
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("connecting to replica %s: %w", url, err)
		}
		if err := pool.Ping(ctx); err != nil {
			return nil, fmt.Errorf("pinging replica %s: %w", url, err)
		}
		pools = append(pools, pool)
	}
	return &ReplicaPool{pools: pools}, nil
}

// Next returns the next replica pool in round-robin order.
func (r *ReplicaPool) Next() *pgxpool.Pool {
	idx := r.counter.Add(1) - 1
	return r.pools[idx%uint64(len(r.pools))]
}

func (r *ReplicaPool) Close() {
	for _, p := range r.pools {
		p.Close()
	}
}
