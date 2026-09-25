package application

import (
	"context"
	"testing"

	"traineebox/internal/auth/domain/errs"

	"github.com/google/uuid"
)

type fakeEnroller struct {
	err    error
	called int
}

func (f *fakeEnroller) EnrollStudent(context.Context, uuid.UUID, bool, uuid.UUID, uuid.UUID) error {
	f.called++
	return f.err
}

func TestProvisionUsers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	groupID := uuid.New()
	actor := uuid.New()

	t.Run("invalid batch", func(t *testing.T) {
		uc := ProvisionUsers{Users: newFakeUsers(), Hasher: PasswordHasher{}, Enroll: &fakeEnroller{}}
		_, err := uc.Execute(ctx, ProvisionInput{ActorID: actor, Admin: true, GroupID: groupID})
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("partial success", func(t *testing.T) {
		users := newFakeUsers()
		uc := ProvisionUsers{Users: users, Hasher: PasswordHasher{}, Enroll: &fakeEnroller{}}
		_, err := uc.Execute(ctx, ProvisionInput{
			ActorID: actor,
			Admin:   true,
			GroupID: groupID,
			Users:   []ProvisionRow{{FullName: "First User", Login: "first.user"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Execute(ctx, ProvisionInput{
			ActorID: actor,
			Admin:   true,
			GroupID: groupID,
			Users: []ProvisionRow{
				{FullName: "Second User", Login: "second.user"},
				{FullName: "First User", Login: "first.user"},
				{FullName: "", Login: "bad"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Created) != 1 || res.Created[0].Password == "" {
			t.Fatalf("created = %+v", res.Created)
		}
		if len(res.Failed) != 2 {
			t.Fatalf("failed = %+v", res.Failed)
		}
		if res.Failed[0].Error != "conflict" {
			t.Fatalf("first fail = %+v", res.Failed[0])
		}
	})
}
