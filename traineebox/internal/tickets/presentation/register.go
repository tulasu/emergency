package presentation

import (
	"context"
	"net/http"

	"traineebox/internal/tickets/application"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-incident-types",
		Method:      http.MethodGet,
		Path:        "/catalog/incident-types",
		Summary:     "List incident types",
		Tags:        []string{"Catalog"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listIncidentTypesHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-incident-tags",
		Method:      http.MethodGet,
		Path:        "/catalog/incident-types/{typeId}/tags",
		Summary:     "List tags for incident type",
		Tags:        []string{"Catalog"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listTagsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-emergency-services",
		Method:      http.MethodGet,
		Path:        "/catalog/services",
		Summary:     "List emergency services",
		Tags:        []string{"Catalog"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listServicesHandler)

	huma.Register(api, huma.Operation{
		OperationID: "create-ticket",
		Method:      http.MethodPost,
		Path:        "/groups/{groupId}/tickets",
		Summary:     "Create ticket in group",
		Tags:        []string{"Tickets"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.createTicketHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-group-tickets",
		Method:      http.MethodGet,
		Path:        "/groups/{groupId}/tickets",
		Summary:     "List tickets in group",
		Tags:        []string{"Tickets"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listTicketsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-ticket",
		Method:      http.MethodGet,
		Path:        "/tickets/{ticketId}",
		Summary:     "Get ticket",
		Tags:        []string{"Tickets"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.getTicketHandler)

	huma.Register(api, huma.Operation{
		OperationID: "set-ticket-reference",
		Method:      http.MethodPut,
		Path:        "/tickets/{ticketId}/reference",
		Summary:     "Set ticket reference answer",
		Tags:        []string{"Tickets"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.setReferenceHandler)

	huma.Register(api, huma.Operation{
		OperationID: "start-ticket-attempt",
		Method:      http.MethodPost,
		Path:        "/tickets/{ticketId}/attempts",
		Summary:     "Start ticket attempt",
		Tags:        []string{"Attempts"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.startAttemptHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-my-attempts",
		Method:      http.MethodGet,
		Path:        "/tickets/{ticketId}/attempts/mine",
		Summary:     "List my attempts for ticket",
		Tags:        []string{"Attempts"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listMyAttemptsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-attempt",
		Method:      http.MethodGet,
		Path:        "/attempts/{attemptId}",
		Summary:     "Get attempt",
		Tags:        []string{"Attempts"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.getAttemptHandler)

	huma.Register(api, huma.Operation{
		OperationID: "save-attempt-answer",
		Method:      http.MethodPatch,
		Path:        "/attempts/{attemptId}/answer",
		Summary:     "Save attempt draft answer",
		Tags:        []string{"Attempts"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.saveAnswerHandler)

	huma.Register(api, huma.Operation{
		OperationID: "submit-attempt",
		Method:      http.MethodPost,
		Path:        "/attempts/{attemptId}/submit",
		Summary:     "Submit attempt",
		Tags:        []string{"Attempts"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.submitAttemptHandler)
}

type authHeader struct {
	Authorization string `header:"Authorization"`
}

func (a *API) listIncidentTypesHandler(ctx context.Context, in *authHeader) (*struct {
	Body []incidentTypeDTO
}, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listIncidentTypes.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]incidentTypeDTO, 0, len(items))
	for _, it := range items {
		out = append(out, incidentTypeDTO{ID: it.ID.String(), Code: it.Code, Title: it.Title})
	}
	return &struct{ Body []incidentTypeDTO }{Body: out}, nil
}

func (a *API) listTagsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TypeID        uuid.UUID `path:"typeId"`
}) (*struct {
	Body []tagGroupDTO
}, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listTagsByType.Execute(ctx, in.TypeID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]tagGroupDTO, 0, len(items))
	for _, g := range items {
		tags := make([]incidentTagDTO, 0, len(g.Tags))
		for _, t := range g.Tags {
			tags = append(tags, incidentTagDTO{
				ID:             t.ID.String(),
				IncidentTypeID: t.IncidentTypeID.String(),
				GroupID:        t.GroupID.String(),
				Code:           t.Code,
				Title:          t.Title,
				SortOrder:      t.SortOrder,
			})
		}
		dto := tagGroupDTO{
			ID:             g.ID.String(),
			IncidentTypeID: g.IncidentTypeID.String(),
			Code:           g.Code,
			Title:          g.Title,
			SelectionMode:  string(g.SelectionMode),
			SortOrder:      g.SortOrder,
			Tags:           tags,
		}
		if g.ParentTagID != nil {
			s := g.ParentTagID.String()
			dto.ParentTagID = &s
		}
		out = append(out, dto)
	}
	return &struct{ Body []tagGroupDTO }{Body: out}, nil
}

func (a *API) listServicesHandler(ctx context.Context, in *authHeader) (*struct {
	Body []serviceDTO
}, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listServices.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]serviceDTO, 0, len(items))
	for _, it := range items {
		out = append(out, serviceDTO{ID: it.ID.String(), Code: it.Code, Title: it.Title})
	}
	return &struct{ Body []serviceDTO }{Body: out}, nil
}

type createTicketInput struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	Body          struct {
		Title           string  `json:"title" minLength:"1" maxLength:"256"`
		Body            string  `json:"body"`
		MaxAttempts     *int    `json:"max_attempts,omitempty"`
		AvailableFrom   *string `json:"available_from,omitempty"`
		AvailableUntil  *string `json:"available_until,omitempty"`
		DurationSeconds *int    `json:"duration_seconds,omitempty"`
	}
}

func (a *API) createTicketHandler(ctx context.Context, in *createTicketInput) (*struct {
	Body ticketDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	from, err := parseOptionalTime(in.Body.AvailableFrom)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	until, err := parseOptionalTime(in.Body.AvailableUntil)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	ticket, err := a.createTicket.Execute(ctx, application.CreateTicketInput{
		ActorID:         user.ID,
		Admin:           isAdmin(user),
		GroupID:         in.GroupID,
		Title:           in.Body.Title,
		Body:            in.Body.Body,
		MaxAttempts:     in.Body.MaxAttempts,
		AvailableFrom:   from,
		AvailableUntil:  until,
		DurationSeconds: in.Body.DurationSeconds,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) listTicketsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
}) (*struct {
	Body []ticketDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	tickets, err := a.listTicketsByGroup.Execute(ctx, application.ListTicketsByGroupInput{
		ActorID: user.ID, Admin: isAdmin(user), GroupID: in.GroupID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]ticketDTO, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, toTicketDTO(t))
	}
	return &struct{ Body []ticketDTO }{Body: out}, nil
}

func (a *API) getTicketHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
}) (*struct {
	Body ticketDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.getTicket.Execute(ctx, application.GetTicketInput{
		ActorID: user.ID, Admin: isAdmin(user), TicketID: in.TicketID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

type setReferenceInput struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
	Body          struct {
		IncidentTypeID     string   `json:"incident_type_id" format:"uuid"`
		TagIDs             []string `json:"tag_ids"`
		ServiceIDs         []string `json:"service_ids"`
		ApplicantLastName  string   `json:"applicant_last_name,omitempty"`
		ApplicantFirstName string   `json:"applicant_first_name,omitempty"`
		CallerNumber       string   `json:"caller_number,omitempty"`
		DictatedNumber     string   `json:"dictated_number,omitempty"`
	}
}

func (a *API) setReferenceHandler(ctx context.Context, in *setReferenceInput) (*struct {
	Body referenceAnswerDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	typeID, err := uuid.Parse(in.Body.IncidentTypeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	tagIDs, err := parseUUIDList(in.Body.TagIDs)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	serviceIDs, err := parseUUIDList(in.Body.ServiceIDs)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	ref, err := a.setReferenceAnswer.Execute(ctx, application.SetReferenceAnswerInput{
		ActorID:            user.ID,
		Admin:              isAdmin(user),
		TicketID:           in.TicketID,
		IncidentTypeID:     typeID,
		TagIDs:             tagIDs,
		ServiceIDs:         serviceIDs,
		ApplicantLastName:  in.Body.ApplicantLastName,
		ApplicantFirstName: in.Body.ApplicantFirstName,
		CallerNumber:       in.Body.CallerNumber,
		DictatedNumber:     in.Body.DictatedNumber,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body referenceAnswerDTO }{Body: toReferenceDTO(ref)}, nil
}

func (a *API) startAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
}) (*struct {
	Body attemptDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.startAttempt.Execute(ctx, application.StartAttemptInput{
		ActorID: user.ID, Admin: isAdmin(user), TicketID: in.TicketID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) listMyAttemptsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
}) (*struct {
	Body []attemptDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempts, err := a.listMyAttempts.Execute(ctx, application.ListMyAttemptsInput{
		ActorID: user.ID, Admin: isAdmin(user), TicketID: in.TicketID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]attemptDTO, 0, len(attempts))
	for _, at := range attempts {
		out = append(out, toAttemptDTO(at))
	}
	return &struct{ Body []attemptDTO }{Body: out}, nil
}

func (a *API) getAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
}) (*struct {
	Body attemptDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.getMyAttempt.Execute(ctx, application.GetMyAttemptInput{
		ActorID: user.ID, Admin: isAdmin(user), AttemptID: in.AttemptID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

type saveAnswerInput struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
	Body          struct {
		IncidentTypeID     *string  `json:"incident_type_id,omitempty"`
		TagIDs             []string `json:"tag_ids"`
		ServiceIDs         []string `json:"service_ids"`
		ApplicantLastName  string   `json:"applicant_last_name"`
		ApplicantFirstName string   `json:"applicant_first_name"`
		CallerNumber       string   `json:"caller_number"`
		DictatedNumber     string   `json:"dictated_number"`
		Notes              string   `json:"notes" maxLength:"1000"`
	}
}

func (a *API) saveAnswerHandler(ctx context.Context, in *saveAnswerInput) (*struct {
	Body attemptDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	typeID, err := parseOptionalUUID(in.Body.IncidentTypeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	tagIDs, err := parseUUIDList(in.Body.TagIDs)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	serviceIDs, err := parseUUIDList(in.Body.ServiceIDs)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid input")
	}
	attempt, err := a.saveAttemptAnswer.Execute(ctx, application.SaveAttemptAnswerInput{
		ActorID:            user.ID,
		Admin:              isAdmin(user),
		AttemptID:          in.AttemptID,
		IncidentTypeID:     typeID,
		TagIDs:             tagIDs,
		ServiceIDs:         serviceIDs,
		ApplicantLastName:  in.Body.ApplicantLastName,
		ApplicantFirstName: in.Body.ApplicantFirstName,
		CallerNumber:       in.Body.CallerNumber,
		DictatedNumber:     in.Body.DictatedNumber,
		Notes:              in.Body.Notes,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) submitAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
}) (*struct {
	Body attemptDTO
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.submitAttempt.Execute(ctx, application.SubmitAttemptInput{
		ActorID: user.ID, Admin: isAdmin(user), AttemptID: in.AttemptID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}
