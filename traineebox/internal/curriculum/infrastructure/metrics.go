package infrastructure

import (
	"context"
	"encoding/json"

	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) VariantMetrics(ctx context.Context, variantID uuid.UUID) (repositories.VariantMetrics, error) {
	var out repositories.VariantMetrics
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM tickets WHERE variant_id = $1
	`, variantID).Scan(&out.TicketCount)
	if err != nil {
		return out, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM ticket_attempts WHERE variant_id = $1
	`, variantID).Scan(&out.AttemptCount)
	if err != nil {
		return out, err
	}
	var avg *float64
	err = s.pool.QueryRow(ctx, `
		SELECT AVG(score)::float8
		FROM ticket_attempts
		WHERE variant_id = $1 AND score IS NOT NULL
	`, variantID).Scan(&avg)
	if err != nil && err != pgx.ErrNoRows {
		return out, err
	}
	out.AvgSuccess = avg

	title, rate, err := s.hardestTicket(ctx, variantID)
	if err != nil {
		return out, err
	}
	out.HardestTicketTitle = title
	out.HardestTicketRate = rate
	if out.HardestTicketTitle == "" && out.TicketCount > 0 {
		_ = s.pool.QueryRow(ctx, `
			SELECT title FROM tickets WHERE variant_id = $1 ORDER BY created_at LIMIT 1
		`, variantID).Scan(&out.HardestTicketTitle)
	}
	return out, nil
}

func (s *Store) hardestTicket(ctx context.Context, variantID uuid.UUID) (string, *float64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT report FROM ticket_attempts
		WHERE variant_id = $1 AND report IS NOT NULL AND report <> 'null'::jsonb
	`, variantID)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()

	type item struct {
		TicketID string `json:"ticket_id"`
		Score    int    `json:"score"`
	}
	type report struct {
		Items []item `json:"items"`
	}
	sums := map[string]struct {
		sum   float64
		count int
	}{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return "", nil, err
		}
		var r report
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		for _, it := range r.Items {
			if it.TicketID == "" {
				continue
			}
			cur := sums[it.TicketID]
			cur.sum += float64(it.Score)
			cur.count++
			sums[it.TicketID] = cur
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	if len(sums) == 0 {
		return "", nil, nil
	}
	var bestID string
	var bestAvg float64
	first := true
	for id, v := range sums {
		if v.count == 0 {
			continue
		}
		avg := v.sum / float64(v.count)
		if first || avg < bestAvg {
			first = false
			bestAvg = avg
			bestID = id
		}
	}
	if bestID == "" {
		return "", nil, nil
	}
	tid, err := uuid.Parse(bestID)
	if err != nil {
		return "", nil, nil
	}
	var title string
	if err := s.pool.QueryRow(ctx, `SELECT title FROM tickets WHERE id = $1`, tid).Scan(&title); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil, nil
		}
		return "", nil, err
	}
	return title, &bestAvg, nil
}

func (s *Store) LessonAttemptStats(ctx context.Context, lessonID uuid.UUID, threshold int) (openedFor int, passedRate *float64, avgSuccess *float64, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT a.user_id)::int
		FROM ticket_attempts a
		JOIN variants v ON v.id = a.variant_id
		WHERE v.lesson_id = $1
	`, lessonID).Scan(&openedFor)
	if err != nil {
		return 0, nil, nil, err
	}
	if openedFor == 0 {
		return 0, nil, nil, nil
	}
	var passed int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM (
			SELECT DISTINCT ON (a.user_id) a.user_id, a.score
			FROM ticket_attempts a
			JOIN variants v ON v.id = a.variant_id
			WHERE v.lesson_id = $1 AND a.score IS NOT NULL
			ORDER BY a.user_id, a.finished_at DESC NULLS LAST, a.attempt_no DESC
		) latest
		WHERE score >= $2
	`, lessonID, threshold).Scan(&passed)
	if err != nil {
		return openedFor, nil, nil, err
	}
	rate := float64(passed) * 100 / float64(openedFor)
	err = s.pool.QueryRow(ctx, `
		SELECT AVG(a.score)::float8
		FROM ticket_attempts a
		JOIN variants v ON v.id = a.variant_id
		WHERE v.lesson_id = $1 AND a.score IS NOT NULL
	`, lessonID).Scan(&avgSuccess)
	if err != nil && err != pgx.ErrNoRows {
		return openedFor, &rate, nil, err
	}
	return openedFor, &rate, avgSuccess, nil
}

func (s *Store) ModuleAttemptStats(ctx context.Context, moduleID uuid.UUID) (openedDone int, successRate *float64, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT v.lesson_id)::int
		FROM ticket_attempts a
		JOIN variants v ON v.id = a.variant_id
		JOIN lessons l ON l.id = v.lesson_id
		WHERE l.module_id = $1 AND l.archived_at IS NULL
	`, moduleID).Scan(&openedDone)
	if err != nil {
		return 0, nil, err
	}
	var avg *float64
	err = s.pool.QueryRow(ctx, `
		SELECT AVG(a.score)::float8
		FROM ticket_attempts a
		JOIN variants v ON v.id = a.variant_id
		JOIN lessons l ON l.id = v.lesson_id
		WHERE l.module_id = $1 AND a.score IS NOT NULL
	`, moduleID).Scan(&avg)
	if err != nil && err != pgx.ErrNoRows {
		return openedDone, nil, err
	}
	return openedDone, avg, nil
}

func (s *Store) AssignmentGroupBreakdown(ctx context.Context, moduleID uuid.UUID) ([]models.AssignmentGroupSummary, int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT source_group_id, COUNT(*)::int
		FROM user_modules
		WHERE module_id = $1
		GROUP BY source_group_id
	`, moduleID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	groups := []models.AssignmentGroupSummary{}
	individuals := 0
	for rows.Next() {
		var gid *uuid.UUID
		var count int
		if err := rows.Scan(&gid, &count); err != nil {
			return nil, 0, err
		}
		if gid == nil {
			individuals = count
			continue
		}
		label := "Группа " + gid.String()[:8]
		var name string
		if err := s.pool.QueryRow(ctx, `SELECT name FROM groups WHERE id = $1`, *gid).Scan(&name); err == nil && name != "" {
			label = name
		}
		groups = append(groups, models.AssignmentGroupSummary{Label: label, Count: count})
	}
	return groups, individuals, rows.Err()
}

func (s *Store) LessonPoolExtras(ctx context.Context, lessonID uuid.UUID) (ticketCount int, variantsLabel string, passedRate *float64, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM tickets t
		JOIN variants v ON v.id = t.variant_id
		WHERE v.lesson_id = $1
	`, lessonID).Scan(&ticketCount)
	if err != nil {
		return 0, "", nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT title, is_primary, status FROM variants WHERE lesson_id = $1 ORDER BY position, created_at
	`, lessonID)
	if err != nil {
		return ticketCount, "", nil, err
	}
	defer rows.Close()
	parts := make([]string, 0)
	for rows.Next() {
		var title, status string
		var primary bool
		if err := rows.Scan(&title, &primary, &status); err != nil {
			return ticketCount, "", nil, err
		}
		label := shortVariantKey(title)
		if primary {
			label += "★"
		}
		if status == "draft" {
			label += " черн."
		}
		parts = append(parts, label)
	}
	if err := rows.Err(); err != nil {
		return ticketCount, "", nil, err
	}
	for i, p := range parts {
		if i == 0 {
			variantsLabel = p
		} else {
			variantsLabel += " · " + p
		}
	}
	_, passedRate, _, err = s.LessonAttemptStats(ctx, lessonID, 70)
	return ticketCount, variantsLabel, passedRate, err
}

func shortVariantKey(title string) string {
	runes := []rune(title)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			if r >= 'a' && r <= 'z' {
				return string(r - 32)
			}
			return string(r)
		}
	}
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if (r >= 'А' && r <= 'Я') || (r >= 'а' && r <= 'я') || r == 'Ё' || r == 'ё' {
			if r >= 'а' && r <= 'я' {
				return string(r - 32)
			}
			if r == 'ё' {
				return "Ё"
			}
			return string(r)
		}
	}
	if len(runes) > 0 {
		return string(runes[0])
	}
	return "?"
}
