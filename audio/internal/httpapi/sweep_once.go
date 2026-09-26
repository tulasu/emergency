// Package httpapi exposes audio service HTTP endpoints.
package httpapi

import (
	"context"
	"time"

	"audio/internal/hash"
	"audio/internal/s3"
	"audio/internal/store"
)

// SweepOnce runs one bounded sweep pass.
func SweepOnce(ctx context.Context, st *store.Store, s3c *s3.Client, ttl time.Duration, limit int) (deleted int64, bytes int64, err error) {
	return st.SweepOrphans(ctx, ttl, limit, func(ctx context.Context, o store.Orphan) error {
		return s3c.Delete(ctx, hash.S3Key(o.Voice, o.Rate, hash.Hex(o.Hash)))
	})
}
