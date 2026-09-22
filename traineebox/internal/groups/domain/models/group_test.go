package models_test

import (
	"errors"
	"testing"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

func TestGroupTransitions(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	teacherID := uuid.New()
	studentID := uuid.New()
	name, err := value_objects.NewGroupName("Cohort A")
	if err != nil {
		t.Fatal(err)
	}
	g := models.NewGroup(name, ownerID)
	if len(g.Members) != 1 || g.Members[0].Role != value_objects.MemberRoleOwner {
		t.Fatalf("owner seed = %+v", g.Members)
	}

	if err := g.AddMember(ownerID, false, teacherID, value_objects.AccountRoleTeacher, value_objects.MemberRoleTeacher); err != nil {
		t.Fatalf("add teacher: %v", err)
	}
	if err := g.AddMember(teacherID, false, studentID, value_objects.AccountRoleStudent, value_objects.MemberRoleStudent); err != nil {
		t.Fatalf("add student: %v", err)
	}
	if err := g.AddMember(teacherID, false, uuid.New(), value_objects.AccountRoleTeacher, value_objects.MemberRoleTeacher); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("co-teacher add teacher: %v", err)
	}
	if err := g.RemoveMember(ownerID, false, ownerID); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("remove last owner: %v", err)
	}

	newName, _ := value_objects.NewGroupName("Cohort B")
	if err := g.Rename(teacherID, false, newName); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("rename by teacher: %v", err)
	}
	if err := g.Rename(ownerID, false, newName); err != nil {
		t.Fatalf("rename by owner: %v", err)
	}
}
