package presentation

import (
	"traineebox/internal/generation/application"
)

type API struct {
	createJob  application.CreateJob
	listJobs   application.ListJobs
	getJob     application.GetJob
	patchJob   application.PatchJob
	retryJob   application.RetryJob
	cancelJob  application.CancelJob
	deleteJob  application.DeleteJob
	approveJob application.ApproveJob
	authenticate application.Authenticator
}

type Deps struct {
	CreateJob    application.CreateJob
	ListJobs     application.ListJobs
	GetJob       application.GetJob
	PatchJob     application.PatchJob
	RetryJob     application.RetryJob
	CancelJob    application.CancelJob
	DeleteJob    application.DeleteJob
	ApproveJob   application.ApproveJob
	Authenticate application.Authenticator
}

func NewAPI(deps Deps) *API {
	return &API{
		createJob:    deps.CreateJob,
		listJobs:     deps.ListJobs,
		getJob:       deps.GetJob,
		patchJob:     deps.PatchJob,
		retryJob:     deps.RetryJob,
		cancelJob:    deps.CancelJob,
		deleteJob:    deps.DeleteJob,
		approveJob:   deps.ApproveJob,
		authenticate: deps.Authenticate,
	}
}
