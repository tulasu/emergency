package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"traineebox/internal/dialog"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"

	"github.com/spf13/cobra"
)

//go:embed all:data/scenarios
var embeddedScenarios embed.FS

// scenario is the minimal shape of a corpus JSON: only id + facts[].slot/questions.
// meta/persona/critical/answers are ignored (ponytail: single consumer, inline struct).
type scenario struct {
	ID    string `json:"id"`
	Facts []struct {
		Slot      string   `json:"slot"`
		Questions []string `json:"questions"`
	} `json:"facts"`
}

type scenarioQuestion struct {
	text   string // original first-seen formulation (display text)
	srcRef string // scenario id it came from
}

func newImportScenariosCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-scenarios",
		Short: "Seed slot_questions from the scenario corpus",
		RunE: func(cmd *cobra.Command, _ []string) error {
			imported, scenarios, skipped, err := importScenarios(cmd.Context(), cfg.PostgresDSN)
			if err != nil {
				return err
			}
			fmt.Printf("imported %d questions from %d scenarios; %d skipped (orphan slots)\n", imported, scenarios, skipped)
			return nil
		},
	}
	return cmd
}

func importScenarios(ctx context.Context, dsn string) (imported, scenarios, skipped int, err error) {
	// bySlot keeps first-seen order per slot; seen dedups on the normalized form
	// (mirrors loader.py: bucket.append(q) stores the raw text, norm only dedups).
	bySlot := map[string][]scenarioQuestion{}
	seen := map[string]map[string]bool{}

	walkErr := fs.WalkDir(embeddedScenarios, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		raw, err := fs.ReadFile(embeddedScenarios, path)
		if err != nil {
			return err
		}
		var sc scenario
		if err := json.Unmarshal(raw, &sc); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		scenarios++
		for _, f := range sc.Facts {
			if f.Slot == "" {
				continue
			}
			slotSeen := seen[f.Slot]
			if slotSeen == nil {
				slotSeen = map[string]bool{}
				seen[f.Slot] = slotSeen
			}
			for _, q := range f.Questions {
				norm := normalizeQuestion(q)
				if norm == "" || slotSeen[norm] {
					continue
				}
				slotSeen[norm] = true
				bySlot[f.Slot] = append(bySlot[f.Slot], scenarioQuestion{text: q, srcRef: sc.ID})
			}
		}
		return nil
	})
	if walkErr != nil {
		return 0, 0, 0, walkErr
	}
	if len(bySlot) == 0 {
		return 0, 0, 0, fmt.Errorf("no questions collected from corpus; refusing to seed")
	}

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		return 0, 0, 0, err
	}
	defer pool.Close()

	known, err := dialog.NewRepository(pool).ListSlotIDs(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	slots := make([]string, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Strings(slots)

	for _, slot := range slots {
		for _, q := range bySlot[slot] {
			if !known[slot] {
				fmt.Fprintf(os.Stderr, "warn: slot %q from %s not in dialog_slots, skipping\n", slot, q.srcRef)
				skipped++
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO slot_questions (slot_id, question, source, source_ref) VALUES ($1, $2, 'legacy', $3)
				 ON CONFLICT (slot_id, question) DO NOTHING`,
				slot, q.text, q.srcRef); err != nil {
				return 0, 0, 0, err
			}
			imported++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, err
	}
	return imported, scenarios, skipped, nil
}

// normalizeQuestion mirrors dispatcher loader.py exactly:
// " ".join(q.lower().replace("ё", "е").split()).strip(" ?.!")
func normalizeQuestion(q string) string {
	s := strings.ToLower(q)
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.Join(strings.Fields(s), " ")
	return strings.Trim(s, " ?.!")
}
