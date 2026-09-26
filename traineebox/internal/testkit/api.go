package testkit

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"traineebox/internal/auth/application"
	autherrs "traineebox/internal/auth/domain/errs"
	authinfra "traineebox/internal/auth/infrastructure"
	authpresentation "traineebox/internal/auth/presentation"
	genapp "traineebox/internal/generation/application"
	generrs "traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/repositories"
	genvo "traineebox/internal/generation/domain/value_objects"
	geninfra "traineebox/internal/generation/infrastructure"
	genpresentation "traineebox/internal/generation/presentation"
	groupsapp "traineebox/internal/groups/application"
	groupserrs "traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
	groupsinfra "traineebox/internal/groups/infrastructure"
	groupspresentation "traineebox/internal/groups/presentation"
	"traineebox/internal/platform/config"
	ticketsapp "traineebox/internal/tickets/application"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"
	ticketsinfra "traineebox/internal/tickets/infrastructure"
	ticketspresentation "traineebox/internal/tickets/presentation"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groupsSessionAuthenticator struct {
	auth application.Authenticate
}

func (a groupsSessionAuthenticator) CurrentUser(ctx context.Context, token string) (groupsapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return groupsapp.SessionUser{}, mapGroupsAuthError(err)
	}
	role, err := value_objects.ParseAccountRole(string(user.Role))
	if err != nil {
		return groupsapp.SessionUser{}, err
	}
	return groupsapp.SessionUser{ID: user.ID, Role: role}, nil
}

type ticketsSessionAuthenticator struct {
	auth application.Authenticate
}

func (a ticketsSessionAuthenticator) CurrentUser(ctx context.Context, token string) (ticketsapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return ticketsapp.SessionUser{}, mapTicketsAuthError(err)
	}
	role, err := ticketsvo.ParseAccountRole(string(user.Role))
	if err != nil {
		return ticketsapp.SessionUser{}, err
	}
	return ticketsapp.SessionUser{ID: user.ID, Role: role}, nil
}

func mapGroupsAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return groupserrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return groupserrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return groupserrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return groupserrs.ErrForbidden
	default:
		return err
	}
}

func mapTicketsAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return ticketserrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return ticketserrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return ticketserrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return ticketserrs.ErrForbidden
	default:
		return err
	}
}

type generationSessionAuthenticator struct {
	auth application.Authenticate
}

func (a generationSessionAuthenticator) CurrentUser(ctx context.Context, token string) (genapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return genapp.SessionUser{}, mapGenerationAuthError(err)
	}
	role, err := genvo.ParseAccountRole(string(user.Role))
	if err != nil {
		return genapp.SessionUser{}, err
	}
	return genapp.SessionUser{ID: user.ID, Role: role}, nil
}

func mapGenerationAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return generrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return generrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return generrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return generrs.ErrForbidden
	default:
		return err
	}
}

func NewAPI(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	users := authinfra.NewUserRepository(pool)
	sessions := authinfra.NewSessionRepository(pool)
	hasher := application.PasswordHasher{}
	authenticate := application.Authenticate{Users: users, Sessions: sessions}

	groupsRepo := groupsinfra.NewGroupRepository(pool)
	directory := groupsinfra.NewUserDirectory(pool)
	addMember := groupsapp.AddMember{Groups: groupsRepo, Directory: directory}

	authHandlers := authpresentation.NewAPI(authpresentation.Deps{
		Version:      "test",
		CreateUser:   application.CreateUser{Users: users, Hasher: hasher},
		Login:        application.Login{Users: users, Sessions: sessions, Hasher: hasher, SessionTTL: time.Hour},
		Logout:       application.Logout{Sessions: sessions},
		Me:           application.Me{Authenticate: authenticate},
		BlockUser:    application.BlockUser{Users: users},
		ChangeRole:   application.ChangeRole{Users: users},
		Authenticate: authenticate,
		Provision: application.ProvisionUsers{
			Users:  users,
			Hasher: hasher,
			Enroll: testStudentEnroller{add: addMember},
		},
	})

	groupsHandlers := groupspresentation.NewAPI(groupspresentation.Deps{
		CreateGroup:  groupsapp.CreateGroup{Groups: groupsRepo, Directory: directory},
		RenameGroup:  groupsapp.RenameGroup{Groups: groupsRepo},
		DeleteGroup:  groupsapp.DeleteGroup{Groups: groupsRepo},
		ListGroups:   groupsapp.ListGroups{Groups: groupsRepo},
		GetGroup:     groupsapp.GetGroup{Groups: groupsRepo},
		AddMember:    addMember,
		RemoveMember: groupsapp.RemoveMember{Groups: groupsRepo},
		Authenticate: groupsSessionAuthenticator{auth: authenticate},
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	cat, err := ticketsinfra.LoadCatalog(cfg.CatalogPath)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	catalogRepo := ticketsinfra.NewCatalogRepository(cat)
	ticketsRepo := ticketsinfra.NewTicketRepository(pool)
	attemptsRepo := ticketsinfra.NewAttemptRepository(pool)
	membership := ticketsinfra.NewGroupMembership(pool)
	createTicketUC := ticketsapp.CreateTicket{Tickets: ticketsRepo, Membership: membership}
	setReferenceUC := ticketsapp.SetReferenceAnswer{Tickets: ticketsRepo, Catalog: catalogRepo, Membership: membership}
	ticketsHandlers := ticketspresentation.NewAPI(ticketspresentation.Deps{
		ListIncidentTypes:  ticketsapp.ListIncidentTypes{Catalog: catalogRepo},
		ListTagsByType:     ticketsapp.ListTagsByType{Catalog: catalogRepo},
		ListServices:       ticketsapp.ListServices{Catalog: catalogRepo},
		RecommendServices:  ticketsapp.RecommendServices{Catalog: catalogRepo},
		CreateTicket:       createTicketUC,
		ListTicketsByGroup: ticketsapp.ListTicketsByGroup{Tickets: ticketsRepo, Membership: membership},
		GetTicket:          ticketsapp.GetTicket{Tickets: ticketsRepo, Membership: membership},
		SetReferenceAnswer: setReferenceUC,
		StartAttempt:       ticketsapp.StartAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		SaveAttemptAnswer:  ticketsapp.SaveAttemptAnswer{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo, Membership: membership},
		SubmitAttempt:      ticketsapp.SubmitAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo, Membership: membership},
		GetMyAttempt:       ticketsapp.GetMyAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		ListMyAttempts:     ticketsapp.ListMyAttempts{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		Authenticate:       ticketsSessionAuthenticator{auth: authenticate},
	})

	jobsRepo := geninfra.NewJobRepository(pool)
	genMembership := testGenerationMembership{inner: membership}
	genCatalog := testGenerationCatalog{tickets: catalogRepo}
	genPublisher := testGenerationPublisher{pool: pool}
	genHandlers := genpresentation.NewAPI(genpresentation.Deps{
		CreateJob: genapp.CreateJob{Jobs: jobsRepo, Membership: genMembership},
		ListJobs:  genapp.ListJobs{Jobs: jobsRepo, Membership: genMembership},
		GetJob:    genapp.GetJob{Jobs: jobsRepo, Membership: genMembership},
		PatchJob:  genapp.PatchJob{Jobs: jobsRepo, Membership: genMembership},
		RetryJob:  genapp.RetryJob{Jobs: jobsRepo, Membership: genMembership},
		CancelJob: genapp.CancelJob{Jobs: jobsRepo, Membership: genMembership},
		DeleteJob: genapp.DeleteJob{Jobs: jobsRepo, Membership: genMembership},
		ApproveJob: genapp.ApproveJob{
			Jobs: jobsRepo, Membership: genMembership, Catalog: genCatalog, Publisher: genPublisher,
		},
		Authenticate: generationSessionAuthenticator{auth: authenticate},
	})

	router := chi.NewMux()
	apiCfg := huma.DefaultConfig("TraineeBox API", "test")
	apiCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "session-token",
		},
	}
	api := humachi.New(router, apiCfg)
	authpresentation.Register(api, authHandlers)
	groupspresentation.Register(api, groupsHandlers)
	ticketspresentation.Register(api, ticketsHandlers)
	genpresentation.Register(api, genHandlers)
	return router
}

type testStudentEnroller struct {
	add groupsapp.AddMember
}

func (e testStudentEnroller) EnrollStudent(ctx context.Context, actorID uuid.UUID, admin bool, groupID, userID uuid.UUID) error {
	_, err := e.add.Execute(ctx, groupsapp.AddMemberInput{
		ActorID: actorID,
		Admin:   admin,
		GroupID: groupID,
		UserID:  userID,
		Role:    "student",
	})
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, groupserrs.ErrForbidden):
		return autherrs.ErrForbidden
	case errors.Is(err, groupserrs.ErrNotFound):
		return autherrs.ErrNotFound
	case errors.Is(err, groupserrs.ErrConflict):
		return autherrs.ErrConflict
	case errors.Is(err, groupserrs.ErrInvalidInput):
		return autherrs.ErrInvalidInput
	case errors.Is(err, groupserrs.ErrUserBlocked):
		return autherrs.ErrUserBlocked
	default:
		return err
	}
}

type testGenerationMembership struct {
	inner ticketsrepos.GroupMembership
}

func (m testGenerationMembership) RoleOf(ctx context.Context, groupID, userID uuid.UUID) (genvo.MemberRole, error) {
	role, err := m.inner.RoleOf(ctx, groupID, userID)
	if err != nil {
		return "", mapTicketsToGeneration(err)
	}
	parsed, err := genvo.ParseMemberRole(role.String())
	if err != nil {
		return "", generrs.ErrInvalidInput
	}
	return parsed, nil
}

type testGenerationCatalog struct {
	tickets ticketsrepos.CatalogRepository
}

func (c testGenerationCatalog) IncidentTypeExists(ctx context.Context, code string) error {
	_, err := c.tickets.FindIncidentTypeByCode(ctx, code)
	return mapTicketsToGeneration(err)
}

func (c testGenerationCatalog) ValidateTags(ctx context.Context, incidentType string, tags []string) error {
	groups, err := c.tickets.ListTagGroupsByType(ctx, incidentType)
	if err != nil {
		return mapTicketsToGeneration(err)
	}
	ref, err := ticketsmodels.NewReferenceAnswer(uuid.New(), incidentType, tags, nil, "", "", "", "")
	if err != nil {
		return mapTicketsToGeneration(err)
	}
	return mapTicketsToGeneration(ref.ValidateTagSelection(groups))
}

func (c testGenerationCatalog) ServicesExist(ctx context.Context, codes []string) (bool, error) {
	ok, err := c.tickets.ServiceExists(ctx, codes)
	return ok, mapTicketsToGeneration(err)
}

type testGenerationPublisher struct {
	pool *pgxpool.Pool
}

func (p testGenerationPublisher) Publish(ctx context.Context, draft repositories.PublishDraft) (uuid.UUID, error) {
	if err := geninfra.ApproveAtomically(ctx, p.pool, draft); err != nil {
		return uuid.Nil, err
	}
	return draft.TicketID, nil
}

func mapTicketsToGeneration(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ticketserrs.ErrForbidden):
		return generrs.ErrForbidden
	case errors.Is(err, ticketserrs.ErrNotFound):
		return generrs.ErrNotFound
	case errors.Is(err, ticketserrs.ErrUnauthorized):
		return generrs.ErrUnauthorized
	case errors.Is(err, ticketserrs.ErrUserBlocked):
		return generrs.ErrUserBlocked
	case errors.Is(err, ticketserrs.ErrInvalidInput), errors.Is(err, ticketserrs.ErrInvalidTags), errors.Is(err, ticketserrs.ErrInvalidTagSelection):
		return generrs.ErrInvalidInput
	case errors.Is(err, ticketserrs.ErrConflict):
		return generrs.ErrConflict
	default:
		return err
	}
}
