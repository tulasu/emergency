package httpapi

import (
	"context"
	"time"

	"audio/internal/repositories"
	"audio/pkg/hash"
	"audio/pkg/s3"
)

// SweepOnce runs one bounded sweep pass.
func SweepOnce(ctx context.Context, st *repositories.Store, s3c *s3.Client, ttl time.Duration, limit int) (deleted int64, bytes int64, err error) {
	return st.SweepOrphans(ctx, ttl, limit, func(ctx context.Context, o repositories.Orphan) error {
		return s3c.Delete(ctx, hash.S3Key(o.Voice, o.Rate, hash.Hex(o.Hash)))
	})
}
