package presentation

import (
	"context"
	"net/http"
	"time"

	"traineebox/internal/tickets/application"
	"traineebox/internal/tickets/domain/errs"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	sec := []map[string][]string{{"session": {}}}

	huma.Register(api, huma.Operation{OperationID: "list-incident-types", Method: http.MethodGet, Path: "/catalog/incident-types", Summary: "List incident types", Tags: []string{"Catalog"}, Security: sec}, a.listIncidentTypesHandler)
	huma.Register(api, huma.Operation{OperationID: "list-incident-tags", Method: http.MethodGet, Path: "/catalog/incident-types/{typeCode}/tags", Summary: "List tags for incident type", Tags: []string{"Catalog"}, Security: sec}, a.listTagsHandler)
	huma.Register(api, huma.Operation{OperationID: "list-emergency-services", Method: http.MethodGet, Path: "/catalog/services", Summary: "List emergency services", Tags: []string{"Catalog"}, Security: sec}, a.listServicesHandler)
	huma.Register(api, huma.Operation{OperationID: "recommend-services", Method: http.MethodGet, Path: "/catalog/recommend", Summary: "Recommend services", Tags: []string{"Catalog"}, Security: sec}, a.recommendServicesHandler)

	huma.Register(api, huma.Operation{OperationID: "list-library-tickets", Method: http.MethodGet, Path: "/tickets", Summary: "List library tickets", Tags: []string{"Tickets"}, Security: sec}, a.listLibraryTicketsHandler)
	huma.Register(api, huma.Operation{OperationID: "create-library-ticket", Method: http.MethodPost, Path: "/tickets", Summary: "Create library ticket", Tags: []string{"Tickets"}, Security: sec}, a.createLibraryTicketHandler)
	huma.Register(api, huma.Operation{OperationID: "create-ticket", Method: http.MethodPost, Path: "/variants/{variantId}/tickets", Summary: "Create ticket in variant", Tags: []string{"Tickets"}, Security: sec}, a.createTicketHandler)
	huma.Register(api, huma.Operation{OperationID: "list-variant-tickets", Method: http.MethodGet, Path: "/variants/{variantId}/tickets", Summary: "List tickets in variant", Tags: []string{"Tickets"}, Security: sec}, a.listTicketsHandler)
	huma.Register(api, huma.Operation{OperationID: "copy-ticket-from-pool", Method: http.MethodPost, Path: "/variants/{variantId}/tickets/from-pool", Summary: "Copy ticket from library pool", Tags: []string{"Tickets"}, Security: sec}, a.copyTicketFromPoolHandler)
	huma.Register(api, huma.Operation{OperationID: "get-ticket", Method: http.MethodGet, Path: "/tickets/{ticketId}", Summary: "Get ticket", Tags: []string{"Tickets"}, Security: sec}, a.getTicketHandler)
	huma.Register(api, huma.Operation{OperationID: "update-ticket", Method: http.MethodPatch, Path: "/tickets/{ticketId}", Summary: "Update ticket", Tags: []string{"Tickets"}, Security: sec}, a.updateTicketHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-ticket", Method: http.MethodDelete, Path: "/tickets/{ticketId}", Summary: "Delete ticket", Tags: []string{"Tickets"}, Security: sec}, a.deleteTicketHandler)
	huma.Register(api, huma.Operation{OperationID: "set-ticket-reference", Method: http.MethodPut, Path: "/tickets/{ticketId}/reference", Summary: "Set ticket reference answer", Tags: []string{"Tickets"}, Security: sec}, a.setReferenceHandler)
	huma.Register(api, huma.Operation{OperationID: "get-ticket-reference", Method: http.MethodGet, Path: "/tickets/{ticketId}/reference", Summary: "Get ticket reference answer", Tags: []string{"Tickets"}, Security: sec}, a.getReferenceHandler)

	huma.Register(api, huma.Operation{OperationID: "grant-attempt", Method: http.MethodPost, Path: "/users/{userId}/attempts", Summary: "Grant attempt to user", Tags: []string{"Attempts"}, Security: sec}, a.grantAttemptHandler)
	huma.Register(api, huma.Operation{OperationID: "open-variant", Method: http.MethodPost, Path: "/variants/{variantId}/open", Summary: "Open variant for users/groups/module", Tags: []string{"Attempts"}, Security: sec}, a.openVariantHandler)
	huma.Register(api, huma.Operation{OperationID: "start-attempt", Method: http.MethodPost, Path: "/attempts/{attemptId}/start", Summary: "Start granted attempt", Tags: []string{"Attempts"}, Security: sec}, a.startAttemptHandler)
	huma.Register(api, huma.Operation{OperationID: "list-my-attempts", Method: http.MethodGet, Path: "/variants/{variantId}/attempts/mine", Summary: "List my attempts for variant", Tags: []string{"Attempts"}, Security: sec}, a.listMyAttemptsHandler)
	huma.Register(api, huma.Operation{OperationID: "list-variant-attempts", Method: http.MethodGet, Path: "/variants/{variantId}/attempts", Summary: "List all attempts for variant", Tags: []string{"Attempts"}, Security: sec}, a.listVariantAttemptsHandler)
	huma.Register(api, huma.Operation{OperationID: "get-attempt", Method: http.MethodGet, Path: "/attempts/{attemptId}", Summary: "Get attempt", Tags: []string{"Attempts"}, Security: sec}, a.getAttemptHandler)
	huma.Register(api, huma.Operation{OperationID: "save-attempt-answer", Method: http.MethodPatch, Path: "/attempts/{attemptId}/answers/{ticketId}", Summary: "Save attempt draft answer", Tags: []string{"Attempts"}, Security: sec}, a.saveAnswerHandler)
	huma.Register(api, huma.Operation{OperationID: "submit-attempt", Method: http.MethodPost, Path: "/attempts/{attemptId}/submit", Summary: "Submit attempt", Tags: []string{"Attempts"}, Security: sec}, a.submitAttemptHandler)
	huma.Register(api, huma.Operation{OperationID: "get-attempt-report", Method: http.MethodGet, Path: "/attempts/{attemptId}/report", Summary: "Get attempt report", Tags: []string{"Attempts"}, Security: sec}, a.getReportHandler)
}

type authHeader struct {
	Authorization string `header:"Authorization"`
}

func (a *API) listIncidentTypesHandler(ctx context.Context, in *authHeader) (*struct{ Body []incidentTypeDTO }, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listIncidentTypes.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]incidentTypeDTO, 0, len(items))
	for _, it := range items {
		out = append(out, incidentTypeDTO{Code: it.Code, Title: it.Title})
	}
	return &struct{ Body []incidentTypeDTO }{Body: out}, nil
}

func (a *API) listTagsHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	TypeCode      string `path:"typeCode"`
}) (*struct{ Body []tagGroupDTO }, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listTagsByType.Execute(ctx, in.TypeCode)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]tagGroupDTO, 0, len(items))
	for _, g := range items {
		tags := make([]incidentTagDTO, 0, len(g.Tags))
		for _, t := range g.Tags {
			tags = append(tags, incidentTagDTO{Code: t.Code, Title: t.Title, SortOrder: t.SortOrder})
		}
		out = append(out, tagGroupDTO{
			Code: g.Code, Title: g.Title, SelectionMode: string(g.SelectionMode),
			ParentTagCode: g.ParentTagCode, SortOrder: g.SortOrder, Tags: tags,
		})
	}
	return &struct{ Body []tagGroupDTO }{Body: out}, nil
}

func (a *API) listServicesHandler(ctx context.Context, in *authHeader) (*struct{ Body []serviceDTO }, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.listServices.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]serviceDTO, 0, len(items))
	for _, it := range items {
		out = append(out, serviceDTO{Code: it.Code, Title: it.Title})
	}
	return &struct{ Body []serviceDTO }{Body: out}, nil
}

func (a *API) recommendServicesHandler(ctx context.Context, in *struct {
	Authorization string   `header:"Authorization"`
	Type          string   `query:"type" required:"true"`
	Tags          []string `query:"tags"`
}) (*struct {
	Body struct {
		ServiceCodes []string `json:"service_codes"`
	}
}, error) {
	if _, err := a.requireSignedIn(ctx, in.Authorization); err != nil {
		return nil, err
	}
	codes, err := a.recommendServices.Execute(ctx, application.RecommendServicesInput{TypeCode: in.Type, TagCodes: in.Tags})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct {
		Body struct {
			ServiceCodes []string `json:"service_codes"`
		}
	}{Body: struct {
		ServiceCodes []string `json:"service_codes"`
	}{ServiceCodes: codes}}, nil
}

func (a *API) listLibraryTicketsHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	Q             string `query:"q"`
}) (*struct{ Body []ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	tickets, err := a.listLibraryTickets.Execute(ctx, user.Role, in.Q)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]ticketDTO, 0, len(tickets))
	for _, t := range tickets {
		dto := toTicketDTO(t)
		dto.CardStatus = cardStatusFromAudio(t.AudioStatus)
		if a.ticketExtras != nil {
			if extras, err := a.ticketExtras.LibraryExtras(ctx, t.ID); err == nil {
				dto.IncidentTypeCode = extras.IncidentTypeCode
				if extras.IncidentTypeCode != "" {
					if extras.IncidentType != "" {
						dto.IncidentType = extras.IncidentTypeCode + " · " + extras.IncidentType
					} else {
						dto.IncidentType = extras.IncidentTypeCode
					}
				}
				st, sr, vu := extras.SlotsTotal, extras.SlotsRequired, extras.VariantUsage
				dto.SlotsTotal = &st
				dto.SlotsRequired = &sr
				dto.VariantUsage = &vu
			}
		}
		out = append(out, dto)
	}
	return &struct{ Body []ticketDTO }{Body: out}, nil
}

func cardStatusFromAudio(audio string) string {
	switch audio {
	case "ready":
		return "ready"
	case "pending":
		return "pending"
	default:
		return "draft"
	}
}

func (a *API) createLibraryTicketHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		TopicID uuid.UUID `json:"topic_id" format:"uuid"`
		Title   string    `json:"title" minLength:"1" maxLength:"256"`
		Body    string    `json:"body"`
	}
}) (*struct{ Body ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.createLibraryTicket.Execute(ctx, application.CreateLibraryTicketInput{
		ActorID: user.ID, Role: user.Role, TopicID: in.Body.TopicID, Title: in.Body.Title, Body: in.Body.Body,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) createTicketHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
	Body          struct {
		TopicID uuid.UUID `json:"topic_id" format:"uuid"`
		Title   string    `json:"title" minLength:"1" maxLength:"256"`
		Body    string    `json:"body"`
	}
}) (*struct{ Body ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.createTicket.Execute(ctx, application.CreateTicketInput{
		ActorID: user.ID, Role: user.Role, VariantID: in.VariantID, TopicID: in.Body.TopicID, Title: in.Body.Title, Body: in.Body.Body,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) copyTicketFromPoolHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
	Body          struct {
		TicketID uuid.UUID `json:"ticket_id" format:"uuid"`
	}
}) (*struct{ Body ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.copyTicketFromPool.Execute(ctx, application.CopyTicketFromPoolInput{
		ActorID: user.ID, Role: user.Role, VariantID: in.VariantID, TicketID: in.Body.TicketID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) listTicketsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
}) (*struct{ Body []ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	tickets, err := a.listTicketsByVariant.Execute(ctx, application.ListTicketsByVariantInput{
		ActorID: user.ID, Role: user.Role, VariantID: in.VariantID,
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
}) (*struct{ Body ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.getTicket.Execute(ctx, application.GetTicketInput{ActorID: user.ID, Role: user.Role, TicketID: in.TicketID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) updateTicketHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
	Body          struct {
		TopicID uuid.UUID `json:"topic_id" format:"uuid"`
		Title   string    `json:"title" minLength:"1" maxLength:"256"`
		Body    string    `json:"body"`
	}
}) (*struct{ Body ticketDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ticket, err := a.updateTicket.Execute(ctx, application.UpdateTicketInput{
		Role: user.Role, TicketID: in.TicketID, TopicID: in.Body.TopicID, Title: in.Body.Title, Body: in.Body.Body,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body ticketDTO }{Body: toTicketDTO(ticket)}, nil
}

func (a *API) deleteTicketHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteTicket.Execute(ctx, user.Role, in.TicketID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) setReferenceHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
	Body          struct {
		IncidentTypeCode   string   `json:"incident_type_code" minLength:"1"`
		TagCodes           []string `json:"tag_codes"`
		ServiceCodes       []string `json:"service_codes"`
		ApplicantLastName  string   `json:"applicant_last_name,omitempty"`
		ApplicantFirstName string   `json:"applicant_first_name,omitempty"`
		CallerNumber       string   `json:"caller_number,omitempty"`
		DictatedNumber     string   `json:"dictated_number,omitempty"`
	}
}) (*struct{ Body referenceAnswerDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ref, err := a.setReferenceAnswer.Execute(ctx, application.SetReferenceAnswerInput{
		Role: user.Role, TicketID: in.TicketID, IncidentTypeCode: in.Body.IncidentTypeCode,
		TagCodes: in.Body.TagCodes, ServiceCodes: in.Body.ServiceCodes,
		ApplicantLastName: in.Body.ApplicantLastName, ApplicantFirstName: in.Body.ApplicantFirstName,
		CallerNumber: in.Body.CallerNumber, DictatedNumber: in.Body.DictatedNumber,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body referenceAnswerDTO }{Body: toReferenceDTO(ref)}, nil
}

func (a *API) getReferenceHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TicketID      uuid.UUID `path:"ticketId"`
}) (*struct{ Body referenceAnswerDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	ref, err := a.getReferenceAnswer.Execute(ctx, user.Role, in.TicketID)
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body referenceAnswerDTO }{Body: toReferenceDTO(ref)}, nil
}

func (a *API) grantAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	UserID        uuid.UUID `path:"userId"`
	Body          struct {
		VariantID     uuid.UUID `json:"variant_id" format:"uuid"`
		AvailableFrom *string   `json:"available_from,omitempty"`
		DeadlineAt    *string   `json:"deadline_at,omitempty"`
	}
}) (*struct{ Body attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	availableFrom, err := parseOptionalRFC3339(in.Body.AvailableFrom)
	if err != nil {
		return nil, mapError(err)
	}
	deadlineAt, err := parseOptionalRFC3339(in.Body.DeadlineAt)
	if err != nil {
		return nil, mapError(err)
	}
	attempt, err := a.grantAttempt.Execute(ctx, application.GrantAttemptInput{
		ActorID: user.ID, Role: user.Role, UserID: in.UserID, VariantID: in.Body.VariantID,
		AvailableFrom: availableFrom, DeadlineAt: deadlineAt,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) openVariantHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
	Body          struct {
		Mode          string      `json:"mode" minLength:"1"`
		ModuleID      *uuid.UUID  `json:"module_id,omitempty" format:"uuid"`
		GroupIDs      []uuid.UUID `json:"group_ids,omitempty"`
		UserIDs       []uuid.UUID `json:"user_ids,omitempty"`
		AvailableFrom *string     `json:"available_from,omitempty"`
		DeadlineAt    *string     `json:"deadline_at,omitempty"`
	}
}) (*struct {
	Body struct {
		Granted int `json:"granted"`
	}
}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	availableFrom, err := parseOptionalRFC3339(in.Body.AvailableFrom)
	if err != nil {
		return nil, mapError(err)
	}
	deadlineAt, err := parseOptionalRFC3339(in.Body.DeadlineAt)
	if err != nil {
		return nil, mapError(err)
	}
	n, err := a.openVariant.Execute(ctx, application.OpenVariantInput{
		ActorID: user.ID, Role: user.Role, VariantID: in.VariantID,
		Mode: in.Body.Mode, ModuleID: in.Body.ModuleID, GroupIDs: in.Body.GroupIDs, UserIDs: in.Body.UserIDs,
		AvailableFrom: availableFrom, DeadlineAt: deadlineAt,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct {
		Body struct {
			Granted int `json:"granted"`
		}
	}{Body: struct {
		Granted int `json:"granted"`
	}{Granted: n}}, nil
}

func parseOptionalRFC3339(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}
	utc := t.UTC()
	return &utc, nil
}

func (a *API) startAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
}) (*struct{ Body attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.startAttempt.Execute(ctx, application.StartAttemptInput{ActorID: user.ID, Role: user.Role, AttemptID: in.AttemptID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) listMyAttemptsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
}) (*struct{ Body []attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempts, err := a.listMyAttempts.Execute(ctx, application.ListMyAttemptsInput{ActorID: user.ID, Role: user.Role, VariantID: in.VariantID})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]attemptDTO, 0, len(attempts))
	for _, at := range attempts {
		out = append(out, toAttemptDTO(at))
	}
	return &struct{ Body []attemptDTO }{Body: out}, nil
}

func (a *API) listVariantAttemptsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
}) (*struct{ Body []attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempts, err := a.listVariantAttempts.Execute(ctx, application.ListVariantAttemptsInput{Role: user.Role, VariantID: in.VariantID})
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
}) (*struct{ Body attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.getMyAttempt.Execute(ctx, application.GetMyAttemptInput{ActorID: user.ID, Role: user.Role, AttemptID: in.AttemptID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) saveAnswerHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
	TicketID      uuid.UUID `path:"ticketId"`
	Body          struct {
		IncidentTypeCode   *string  `json:"incident_type_code,omitempty"`
		TagCodes           []string `json:"tag_codes"`
		ServiceCodes       []string `json:"service_codes"`
		ApplicantLastName  string   `json:"applicant_last_name"`
		ApplicantFirstName string   `json:"applicant_first_name"`
		CallerNumber       string   `json:"caller_number"`
		DictatedNumber     string   `json:"dictated_number"`
		Notes              string   `json:"notes" maxLength:"1000"`
	}
}) (*struct{ Body attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.saveAttemptAnswer.Execute(ctx, application.SaveAttemptAnswerInput{
		ActorID: user.ID, Role: user.Role, AttemptID: in.AttemptID, TicketID: in.TicketID,
		IncidentTypeCode: in.Body.IncidentTypeCode, TagCodes: in.Body.TagCodes, ServiceCodes: in.Body.ServiceCodes,
		ApplicantLastName: in.Body.ApplicantLastName, ApplicantFirstName: in.Body.ApplicantFirstName,
		CallerNumber: in.Body.CallerNumber, DictatedNumber: in.Body.DictatedNumber, Notes: in.Body.Notes,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) submitAttemptHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
}) (*struct{ Body attemptDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.submitAttempt.Execute(ctx, application.SubmitAttemptInput{ActorID: user.ID, Role: user.Role, AttemptID: in.AttemptID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attemptDTO }{Body: toAttemptDTO(attempt)}, nil
}

func (a *API) getReportHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttemptID     uuid.UUID `path:"attemptId"`
}) (*struct{ Body reportDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	attempt, err := a.getAttemptReport.Execute(ctx, user.ID, user.Role, in.AttemptID)
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body reportDTO }{Body: toReportDTO(attempt.Report)}, nil
}
