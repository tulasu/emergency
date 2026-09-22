package models

import (
	"time"

	"traineebox/internal/groups/domain/abilities"
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

type Member struct {
	UserID   uuid.UUID
	Role     value_objects.MemberRole
	JoinedAt time.Time
}

type Group struct {
	ID        uuid.UUID
	Name      value_objects.GroupName
	CreatedAt time.Time
	Members   []Member
}

func NewGroup(name value_objects.GroupName, ownerID uuid.UUID) Group {
	now := time.Now().UTC()
	return Group{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		Members: []Member{{
			UserID:   ownerID,
			Role:     value_objects.MemberRoleOwner,
			JoinedAt: now,
		}},
	}
}

func (g *Group) AddMember(
	actorID uuid.UUID,
	admin bool,
	targetID uuid.UUID,
	account value_objects.AccountRole,
	role value_objects.MemberRole,
) error {
	actorRole, err := g.actorRole(actorID, admin)
	if err != nil {
		return err
	}
	if err := abilities.AddMember(actorRole, admin, account, role, g.hasMember(targetID)); err != nil {
		return err
	}
	g.Members = append(g.Members, Member{
		UserID:   targetID,
		Role:     role,
		JoinedAt: time.Now().UTC(),
	})
	return nil
}

func (g *Group) RemoveMember(actorID uuid.UUID, admin bool, targetID uuid.UUID) error {
	target, ok := g.findMember(targetID)
	if !ok {
		return errs.ErrNotFound
	}
	actorRole, err := g.actorRole(actorID, admin)
	if err != nil {
		return err
	}
	if err := abilities.RemoveMember(actorRole, admin, target.Role, g.ownerCount()); err != nil {
		return err
	}
	members := make([]Member, 0, len(g.Members)-1)
	for _, m := range g.Members {
		if m.UserID != targetID {
			members = append(members, m)
		}
	}
	g.Members = members
	return nil
}

func (g *Group) Rename(actorID uuid.UUID, admin bool, name value_objects.GroupName) error {
	actorRole, err := g.actorRole(actorID, admin)
	if err != nil {
		return err
	}
	if err := abilities.RenameGroup(actorRole, admin); err != nil {
		return err
	}
	g.Name = name
	return nil
}

func (g *Group) Delete(actorID uuid.UUID, admin bool) error {
	actorRole, err := g.actorRole(actorID, admin)
	if err != nil {
		return err
	}
	return abilities.DeleteGroup(actorRole, admin)
}

func (g Group) MemberOf(userID uuid.UUID) (Member, bool) {
	return g.findMember(userID)
}

func (g Group) actorRole(actorID uuid.UUID, admin bool) (value_objects.MemberRole, error) {
	if admin {
		return "", nil
	}
	m, ok := g.findMember(actorID)
	if !ok {
		return "", errs.ErrForbidden
	}
	return m.Role, nil
}

func (g Group) findMember(userID uuid.UUID) (Member, bool) {
	for _, m := range g.Members {
		if m.UserID == userID {
			return m, true
		}
	}
	return Member{}, false
}

func (g Group) hasMember(userID uuid.UUID) bool {
	_, ok := g.findMember(userID)
	return ok
}

func (g Group) ownerCount() int {
	n := 0
	for _, m := range g.Members {
		if m.Role == value_objects.MemberRoleOwner {
			n++
		}
	}
	return n
}
