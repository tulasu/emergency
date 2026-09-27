package value_objects

import "traineebox/internal/curriculum/domain/errs"

type ModuleStatus string

const (
	ModuleStatusDraft    ModuleStatus = "draft"
	ModuleStatusActive   ModuleStatus = "active"
	ModuleStatusArchived ModuleStatus = "archived"
)

func NewModuleStatus(s string) (ModuleStatus, error) {
	switch ModuleStatus(s) {
	case ModuleStatusDraft, ModuleStatusActive, ModuleStatusArchived:
		return ModuleStatus(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (s ModuleStatus) String() string { return string(s) }

type VariantStatus string

const (
	VariantStatusDraft    VariantStatus = "draft"
	VariantStatusApproved VariantStatus = "approved"
)

func NewVariantStatus(s string) (VariantStatus, error) {
	switch VariantStatus(s) {
	case VariantStatusDraft, VariantStatusApproved:
		return VariantStatus(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (s VariantStatus) String() string { return string(s) }
