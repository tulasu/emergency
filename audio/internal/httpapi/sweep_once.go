// SweepOnce deletes up to limit orphan blobs past TTL: S3 keys first,
// then DB rows (S3-first so a crash leaves a DB row the next sweep
// retries, never a dangling reference).
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
	orphans, err := st.SweepOrphans(ctx, ttl, limit)
	if err != nil {
		return 0, 0, err
	}
	done := [][32]byte{}
	for _, o := range orphans {
		key := hash.S3Key(o.Voice, o.Rate, hash.Hex(o.Hash))
		if err := s3c.Delete(ctx, key); err != nil {
			continue // next sweep retries; row stays
		}
		done = append(done, o.Hash)
		bytes += int64(o.Bytes)
	}
	if len(done) == 0 {
		return 0, 0, nil
	}
	deleted, err = st.DeleteBlobs(ctx, done)
	return deleted, bytes, err
}
