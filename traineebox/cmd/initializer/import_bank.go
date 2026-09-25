package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed data/slots.yaml
var embeddedSlots []byte

type slotFile struct {
	Version   string            `yaml:"version"`
	Slots     []slotDef         `yaml:"slots"`
	Overrides map[string]string `yaml:"overrides"`
}

type slotDef struct {
	ID       string   `yaml:"id"`
	Label    string   `yaml:"label"`
	Kind     string   `yaml:"kind"`
	Urge     string   `yaml:"urge"`
	Fallback []string `yaml:"fallback"`
	Aliases  []string `yaml:"aliases"`
}

func newImportBankCmd(cfg config.Config) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "import-bank",
		Short: "Seed slot ontology from embedded slots.yaml",
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw := embeddedSlots
			if file != "" {
				b, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				raw = b
			}
			n, m, version, digest, err := importBank(cmd.Context(), cfg.PostgresDSN, raw)
			if err != nil {
				return err
			}
			fmt.Printf("imported %d slots, %d questions; bank version=%s digest=%s\n", n, m, version, digest)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "override embedded slots.yaml (for tests/CI)")
	return cmd
}

func importBank(ctx context.Context, dsn string, raw []byte) (slots, questions int, version, digest string, err error) {
	var f slotFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return 0, 0, "", "", fmt.Errorf("parse slots.yaml: %w", err)
	}
	if len(f.Slots) == 0 {
		return 0, 0, "", "", fmt.Errorf("slots.yaml parsed but yields zero slots; refusing to seed")
	}

	sum := sha256.Sum256(raw)
	digest = fmt.Sprintf("%x", sum)[:16]

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		return 0, 0, "", "", err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, s := range f.Slots {
		kind := s.Kind
		if kind == "" {
			kind = "value"
		}
		// Family = id prefix before first '.', matching the derivation in
		// ReplaceBank (internal/dialog/repository.go).
		family := strings.SplitN(s.ID, ".", 2)[0]
		if _, err := tx.Exec(ctx,
			`INSERT INTO dialog_slots (id, label, kind, family) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, kind = EXCLUDED.kind, family = EXCLUDED.family`,
			s.ID, s.Label, kind, family); err != nil {
			return 0, 0, "", "", err
		}
		if s.Urge != "" {
			if _, err := tx.Exec(ctx,
				`INSERT INTO slot_questions (slot_id, question, source) VALUES ($1, $2, 'urge')
				 ON CONFLICT (slot_id, question) DO NOTHING`,
				s.ID, s.Urge); err != nil {
				return 0, 0, "", "", err
			}
			questions++
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO bank_state (version, digest) VALUES ($1, $2)
		 ON CONFLICT (version) DO UPDATE SET digest = EXCLUDED.digest, updated_at = now()`,
		f.Version, digest); err != nil {
		return 0, 0, "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, "", "", err
	}
	return len(f.Slots), questions, f.Version, digest, nil
}
