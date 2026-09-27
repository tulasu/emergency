package repositories

import "context"

type BankRepository interface {
	ListSlotIDs(ctx context.Context) (map[string]bool, error)
	ListSlotLabels(ctx context.Context) (map[string]string, error)
	ListSlotUrges(ctx context.Context) (map[string]string, error)
	ReplaceBank(ctx context.Context, version string, slots map[string]string, questions map[string][]string) error
	BankSnapshot(ctx context.Context) (slots map[string]string, questions map[string][]string, err error)
	BankVersion(ctx context.Context) (version, digest string, err error)
}
