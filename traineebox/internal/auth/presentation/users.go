package presentation

import (
	"context"

	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type userProfileDTO struct {
	ID        string  `json:"id"`
	Login     string  `json:"login"`
	Role      string  `json:"role"`
	FullName  string  `json:"full_name"`
	Blocked   bool    `json:"blocked"`
	BlockedAt *string `json:"blocked_at,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
}

func toUserProfileDTO(u models.User) userProfileDTO {
	dto := userProfileDTO{
		ID:       u.ID.String(),
		Login:    u.Login.String(),
		Role:     string(u.Role),
		FullName: u.FullName,
		Blocked:  u.IsBlocked(),
	}
	if u.BlockedAt != nil {
		s := u.BlockedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		dto.BlockedAt = &s
	}
	if !u.CreatedAt.IsZero() {
		dto.CreatedAt = u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return dto
}

func (a *API) listUsersHandler(ctx context.Context, in *authHeaderInput) (*struct{ Body []userProfileDTO }, error) {
	if _, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin, value_objects.RoleTeacher); err != nil {
		return nil, err
	}
	users, err := a.listUsers.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]userProfileDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserProfileDTO(u))
	}
	return &struct{ Body []userProfileDTO }{Body: out}, nil
}

func (a *API) getUserHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	UserID        uuid.UUID `path:"userId"`
}) (*struct{ Body userProfileDTO }, error) {
	if _, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin, value_objects.RoleTeacher); err != nil {
		return nil, err
	}
	user, err := a.getUser.Execute(ctx, in.UserID)
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body userProfileDTO }{Body: toUserProfileDTO(user)}, nil
}

func (a *API) changePasswordHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		CurrentPassword string `json:"current_password" minLength:"1"`
		NewPassword     string `json:"new_password" minLength:"8" maxLength:"128"`
	}
}) (*emptyOutput, error) {
	user, err := a.authenticate.Execute(ctx, bearerToken(in.Authorization))
	if err != nil {
		return nil, mapError(err)
	}
	if err := a.changePassword.Execute(ctx, user.ID, in.Body.CurrentPassword, in.Body.NewPassword); err != nil {
		return nil, mapError(err)
	}
	return &emptyOutput{}, nil
}
