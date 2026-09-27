package repositories

import (
	"context"
	"fmt"
)

// ReadyManifestsForHash returns every manifest made ready by a completed
// synthesis hash. A shared queue item can satisfy multiple ticket manifests.
func (s *Store) ReadyManifestsForHash(ctx context.Context, sum [32]byte) ([]Manifest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ticket_id, scenario_digest
		FROM manifests
		WHERE status = 'ready'
		  AND EXISTS (
			SELECT 1 FROM jsonb_each_text(fragments) e WHERE e.value = $1
		  )
		ORDER BY ticket_id`, hexOf(sum))
	if err != nil {
		return nil, fmt.Errorf("ready manifests: %w", err)
	}
	defer rows.Close()

	manifests := []Manifest{}
	for rows.Next() {
		var manifest Manifest
		if err := rows.Scan(&manifest.TicketID, &manifest.Digest); err != nil {
			return nil, fmt.Errorf("scan ready manifest: %w", err)
		}
		manifest.Status = "ready"
		manifests = append(manifests, manifest)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ready manifests: %w", err)
	}
	return manifests, nil
}
