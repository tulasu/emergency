package presentation

import (
	"context"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type provisionUserRow struct {
	FullName string `json:"full_name" minLength:"1" maxLength:"128"`
	Login    string `json:"login" minLength:"3" maxLength:"64"`
}

type provisionInput struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		GroupID string             `json:"group_id" format:"uuid"`
		Users   []provisionUserRow `json:"users" minItems:"1" maxItems:"200"`
	}
}

type provisionedDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type provisionFailedDTO struct {
	Index int    `json:"index"`
	Login string `json:"login"`
	Error string `json:"error"`
}

type provisionOutput struct {
	Body struct {
		Created []provisionedDTO     `json:"created"`
		Failed  []provisionFailedDTO `json:"failed"`
	}
}

func (a *API) provisionUsersHandler(ctx context.Context, in *provisionInput) (*provisionOutput, error) {
	actor, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin, value_objects.RoleTeacher)
	if err != nil {
		return nil, err
	}
	groupID, err := uuid.Parse(in.Body.GroupID)
	if err != nil {
		return nil, huma.Error400BadRequest(codeInvalidInput)
	}
	rows := make([]application.ProvisionRow, 0, len(in.Body.Users))
	for _, u := range in.Body.Users {
		rows = append(rows, application.ProvisionRow{FullName: u.FullName, Login: u.Login})
	}
	res, err := a.provision.Execute(ctx, application.ProvisionInput{
		ActorID: actor.ID,
		Admin:   actor.Role == value_objects.RoleAdmin,
		GroupID: groupID,
		Users:   rows,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := &provisionOutput{}
	out.Body.Created = make([]provisionedDTO, 0, len(res.Created))
	for _, c := range res.Created {
		out.Body.Created = append(out.Body.Created, provisionedDTO{
			ID:       c.User.ID.String(),
			FullName: c.User.FullName,
			Login:    c.User.Login.String(),
			Password: c.Password,
		})
	}
	out.Body.Failed = make([]provisionFailedDTO, 0, len(res.Failed))
	for _, f := range res.Failed {
		out.Body.Failed = append(out.Body.Failed, provisionFailedDTO{
			Index: f.Index,
			Login: f.Login,
			Error: f.Error,
		})
	}
	return out, nil
}
