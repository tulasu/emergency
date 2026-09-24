package value_objects

import "traineebox/internal/generation/domain/errs"

type JobStatus string

const (
	JobStatusQueued           JobStatus = "queued"
	JobStatusEnriching        JobStatus = "enriching"
	JobStatusFillingPII       JobStatus = "filling_pii"
	JobStatusPickingType      JobStatus = "picking_type"
	JobStatusTaggingType      JobStatus = "tagging_type"
	JobStatusTaggingCommon    JobStatus = "tagging_common"
	JobStatusBuildingServices JobStatus = "building_services"
	JobStatusBuildingDialog   JobStatus = "building_dialog"
	JobStatusCheckingDialog   JobStatus = "checking_dialog"
	JobStatusBuildingRef      JobStatus = "building_ref" // legacy mid-pipeline
	JobStatusReady            JobStatus = "ready"
	JobStatusFailed           JobStatus = "failed"
	JobStatusCancelled        JobStatus = "cancelled"
	JobStatusPublished        JobStatus = "published"
)

func ParseJobStatus(s string) (JobStatus, error) {
	switch JobStatus(s) {
	case JobStatusQueued, JobStatusEnriching, JobStatusFillingPII,
		JobStatusPickingType, JobStatusTaggingType, JobStatusTaggingCommon,
		JobStatusBuildingServices, JobStatusBuildingDialog, JobStatusCheckingDialog,
		JobStatusBuildingRef,
		JobStatusReady, JobStatusFailed, JobStatusCancelled, JobStatusPublished:
		return JobStatus(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (s JobStatus) String() string { return string(s) }

func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCancelled || s == JobStatusPublished
}

func (s JobStatus) CanEditDraft() bool {
	return s == JobStatusReady
}

func (s JobStatus) CanRetry() bool {
	return s == JobStatusFailed
}

func (s JobStatus) IsInProgress() bool {
	switch s {
	case JobStatusEnriching, JobStatusFillingPII, JobStatusPickingType,
		JobStatusTaggingType, JobStatusTaggingCommon, JobStatusBuildingServices,
		JobStatusBuildingDialog, JobStatusCheckingDialog, JobStatusBuildingRef:
		return true
	default:
		return false
	}
}

func (s JobStatus) CanCancel() bool {
	return s == JobStatusQueued || s == JobStatusReady || s == JobStatusFailed || s.IsInProgress()
}

func (s JobStatus) CanApprove() bool {
	return s == JobStatusReady
}

func (s JobStatus) CanDelete() bool {
	return s != JobStatusPublished
}
