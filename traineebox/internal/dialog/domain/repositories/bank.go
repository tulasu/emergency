package repositories

import "context"

type BankRepository interface {
	ListSlotIDs(ctx context.Context) (map[string]bool, error)
	ReplaceBank(ctx context.Context, version string, slots map[string]string, questions map[string][]string) error
	BankSnapshot(ctx context.Context) (slots map[string]string, questions map[string][]string, err error)
	BankVersion(ctx context.Context) (version, digest string, err error)
}
