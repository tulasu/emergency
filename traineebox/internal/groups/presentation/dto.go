package presentation

import (
	"traineebox/internal/groups/domain/models"
)

type memberDTO struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type groupDTO struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Members []memberDTO `json:"members"`
}

func toGroupDTO(g models.Group) groupDTO {
	members := make([]memberDTO, 0, len(g.Members))
	for _, m := range g.Members {
		members = append(members, memberDTO{
			UserID: m.UserID.String(),
			Role:   m.Role.String(),
		})
	}
	return groupDTO{
		ID:      g.ID.String(),
		Name:    g.Name.String(),
		Members: members,
	}
}

type emptyOutput struct {
	Body struct{}
}
