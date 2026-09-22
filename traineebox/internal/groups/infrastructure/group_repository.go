package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/value_objects"
	"traineebox/internal/groups/infrastructure/groupssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository struct {
	pool *pgxpool.Pool
	q    *groupssql.Queries
}

func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool, q: groupssql.New(pool)}
}

func (r *GroupRepository) Create(ctx context.Context, group models.Group) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	if err := q.CreateGroup(ctx, groupssql.CreateGroupParams{
		ID:        group.ID,
		Name:      group.Name.String(),
		CreatedAt: group.CreatedAt,
	}); err != nil {
		return err
	}
	for _, m := range group.Members {
		if err := q.CreateGroupMember(ctx, groupssql.CreateGroupMemberParams{
			GroupID:    group.ID,
			UserID:     m.UserID,
			MemberRole: m.Role.String(),
			JoinedAt:   m.JoinedAt,
		}); err != nil {
			if isUniqueViolation(err) {
				return errs.ErrConflict
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *GroupRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Group, error) {
	row, err := r.q.GetGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Group{}, errs.ErrNotFound
		}
		return models.Group{}, err
	}
	members, err := r.q.ListGroupMembers(ctx, id)
	if err != nil {
		return models.Group{}, err
	}
	return mapGroup(row, members), nil
}

func (r *GroupRepository) ListAll(ctx context.Context) ([]models.Group, error) {
	rows, err := r.q.ListAllGroups(ctx)
	if err != nil {
		return nil, err
	}
	return r.loadGroups(ctx, rows)
}

func (r *GroupRepository) ListByMember(ctx context.Context, userID uuid.UUID) ([]models.Group, error) {
	ids, err := r.q.ListGroupIDsByMember(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups := make([]models.Group, 0, len(ids))
	for _, id := range ids {
		g, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *GroupRepository) UpdateName(ctx context.Context, id uuid.UUID, name value_objects.GroupName) error {
	return r.q.UpdateGroupName(ctx, groupssql.UpdateGroupNameParams{
		ID:   id,
		Name: name.String(),
	})
}

func (r *GroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteGroup(ctx, id)
}

func (r *GroupRepository) AddMember(ctx context.Context, groupID uuid.UUID, member models.Member) error {
	err := r.q.CreateGroupMember(ctx, groupssql.CreateGroupMemberParams{
		GroupID:    groupID,
		UserID:     member.UserID,
		MemberRole: member.Role.String(),
		JoinedAt:   member.JoinedAt,
	})
	if isUniqueViolation(err) {
		return errs.ErrConflict
	}
	return err
}

func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	return r.q.DeleteGroupMember(ctx, groupssql.DeleteGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	})
}

func (r *GroupRepository) loadGroups(ctx context.Context, rows []groupssql.Group) ([]models.Group, error) {
	groups := make([]models.Group, 0, len(rows))
	for _, row := range rows {
		members, err := r.q.ListGroupMembers(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		groups = append(groups, mapGroup(row, members))
	}
	return groups, nil
}

func mapGroup(row groupssql.Group, members []groupssql.GroupMember) models.Group {
	out := models.Group{
		ID:        row.ID,
		Name:      value_objects.GroupName(row.Name),
		CreatedAt: row.CreatedAt,
		Members:   make([]models.Member, 0, len(members)),
	}
	for _, m := range members {
		out.Members = append(out.Members, models.Member{
			UserID:   m.UserID,
			Role:     value_objects.MemberRole(m.MemberRole),
			JoinedAt: m.JoinedAt,
		})
	}
	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
