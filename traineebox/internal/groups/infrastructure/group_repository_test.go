package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"traineebox/internal/auth/domain/value_objects"
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	groupsvo "traineebox/internal/groups/domain/value_objects"
	"traineebox/internal/groups/infrastructure"
	"traineebox/internal/testkit"
)

func TestGroupRepositoryAndDirectory(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)

	owner := testkit.SeedUser(t, pool, "gowner", "password1", value_objects.RoleTeacher)
	student := testkit.SeedUser(t, pool, "gstud", "password1", value_objects.RoleStudent)

	dir := infrastructure.NewUserDirectory(pool)
	role, blocked, err := dir.AccountOf(context.Background(), owner.ID)
	if err != nil || blocked || role != groupsvo.AccountRoleTeacher {
		t.Fatalf("account = %s blocked=%v err=%v", role, blocked, err)
	}

	repo := infrastructure.NewGroupRepository(pool)
	name, _ := groupsvo.NewGroupName("Lab")
	group := models.NewGroup(name, owner.ID)
	if err := repo.Create(context.Background(), group); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByID(context.Background(), group.ID)
	if err != nil || got.Name.String() != "Lab" || len(got.Members) != 1 {
		t.Fatalf("find = %+v err=%v", got, err)
	}

	member := models.Member{
		UserID:   student.ID,
		Role:     groupsvo.MemberRoleStudent,
		JoinedAt: time.Now().UTC(),
	}
	if err := repo.AddMember(context.Background(), group.ID, member); err != nil {
		t.Fatalf("add member: %v", err)
	}
	if err := repo.AddMember(context.Background(), group.ID, member); err != errs.ErrConflict {
		t.Fatalf("dup member err = %v", err)
	}

	listed, err := repo.ListByMember(context.Background(), student.ID)
	if err != nil || len(listed) != 1 || listed[0].ID != group.ID {
		t.Fatalf("list by member = %+v err=%v", listed, err)
	}

	if err := repo.RemoveMember(context.Background(), group.ID, student.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := repo.Delete(context.Background(), group.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), group.ID); err != errs.ErrNotFound {
		t.Fatalf("after delete err = %v", err)
	}
}
