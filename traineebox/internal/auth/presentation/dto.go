package presentation

import (
	"traineebox/internal/auth/domain/models"
)

type userDTO struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
}

func toUserDTO(u models.User) userDTO {
	return userDTO{
		ID:       u.ID.String(),
		Login:    u.Login.String(),
		Role:     string(u.Role),
		FullName: u.FullName,
	}
}

type emptyOutput struct {
	Body struct{}
}

type authHeaderInput struct {
	Authorization string `header:"Authorization"`
}
