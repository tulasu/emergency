package presentation

import (
	"context"
	"net/http"

	"traineebox/internal/generation/application"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-generation-job",
		Method:      http.MethodPost,
		Path:        "/groups/{groupId}/generation-jobs",
		Summary:     "Create ticket generation job",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.createJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-generation-jobs",
		Method:      http.MethodGet,
		Path:        "/groups/{groupId}/generation-jobs",
		Summary:     "List ticket generation jobs",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listJobsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-generation-job",
		Method:      http.MethodGet,
		Path:        "/generation-jobs/{jobId}",
		Summary:     "Get generation job",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.getJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "patch-generation-job",
		Method:      http.MethodPatch,
		Path:        "/generation-jobs/{jobId}",
		Summary:     "Edit generation job draft",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.patchJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "retry-generation-job",
		Method:      http.MethodPost,
		Path:        "/generation-jobs/{jobId}/retry",
		Summary:     "Retry failed generation job",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.retryJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "cancel-generation-job",
		Method:      http.MethodPost,
		Path:        "/generation-jobs/{jobId}/cancel",
		Summary:     "Cancel generation job",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.cancelJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "delete-generation-job",
		Method:      http.MethodDelete,
		Path:        "/generation-jobs/{jobId}",
		Summary:     "Delete generation job",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.deleteJobHandler)

	huma.Register(api, huma.Operation{
		OperationID: "approve-generation-job",
		Method:      http.MethodPost,
		Path:        "/generation-jobs/{jobId}/approve",
		Summary:     "Approve generation job and publish ticket",
		Tags:        []string{"Generation"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.approveJobHandler)
}

func (a *API) createJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	Body          struct {
		Prompt string `json:"prompt" minLength:"1"`
	}
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.createJob.Execute(ctx, application.CreateJobInput{
		ActorID: user.ID, Admin: isAdmin(user), GroupID: in.GroupID, Prompt: in.Body.Prompt,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}

func (a *API) listJobsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
}) (*struct {
	Body []jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	jobs, err := a.listJobs.Execute(ctx, application.ListJobsInput{
		ActorID: user.ID, Admin: isAdmin(user), GroupID: in.GroupID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]jobDTO, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toJobDTO(j))
	}
	return &struct{ Body []jobDTO }{Body: out}, nil
}

func (a *API) getJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.getJob.Execute(ctx, application.GetJobInput{
		ActorID: user.ID, Admin: isAdmin(user), JobID: in.JobID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}

func (a *API) patchJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
	Body          struct {
		DraftTitle   string            `json:"draft_title" minLength:"1" maxLength:"256"`
		ScenarioText string            `json:"scenario_text"`
		Reference    draftReferenceDTO `json:"draft_reference"`
	}
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.patchJob.Execute(ctx, application.PatchJobInput{
		ActorID:      user.ID,
		Admin:        isAdmin(user),
		JobID:        in.JobID,
		DraftTitle:   in.Body.DraftTitle,
		ScenarioText: in.Body.ScenarioText,
		Reference:    fromDraftDTO(in.Body.Reference),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}

func (a *API) retryJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.retryJob.Execute(ctx, application.RetryJobInput{
		ActorID: user.ID, Admin: isAdmin(user), JobID: in.JobID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}

func (a *API) cancelJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.cancelJob.Execute(ctx, application.CancelJobInput{
		ActorID: user.ID, Admin: isAdmin(user), JobID: in.JobID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}

func (a *API) deleteJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteJob.Execute(ctx, application.DeleteJobInput{
		ActorID: user.ID, Admin: isAdmin(user), JobID: in.JobID,
	}); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) approveJobHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	JobID         uuid.UUID `path:"jobId"`
}) (*struct {
	Body jobDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	job, err := a.approveJob.Execute(ctx, application.ApproveJobInput{
		ActorID: user.ID, Admin: isAdmin(user), JobID: in.JobID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body jobDTO }{Body: toJobDTO(job)}, nil
}
