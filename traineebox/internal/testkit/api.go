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
	currapp "traineebox/internal/curriculum/application"
	currerrs "traineebox/internal/curriculum/domain/errs"
	currvo "traineebox/internal/curriculum/domain/value_objects"
	currinfra "traineebox/internal/curriculum/infrastructure"
	currpresentation "traineebox/internal/curriculum/presentation"
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

type curriculumSessionAuthenticator struct {
	auth application.Authenticate
}

func (a curriculumSessionAuthenticator) CurrentUser(ctx context.Context, token string) (currapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return currapp.SessionUser{}, mapCurriculumAuthError(err)
	}
	role, err := currvo.ParseAccountRole(string(user.Role))
	if err != nil {
		return currapp.SessionUser{}, err
	}
	return currapp.SessionUser{ID: user.ID, Role: role}, nil
}

func mapCurriculumAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return currerrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return currerrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return currerrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return currerrs.ErrForbidden
	default:
		return err
	}
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
	currStore := currinfra.NewStore(pool)
	topics := currinfra.Topics{Store: currStore}
	articles := currinfra.Articles{Store: currStore}
	attachments := currinfra.Attachments{Store: currStore}
	modules := currinfra.Modules{Store: currStore}
	lessons := currinfra.Lessons{Store: currStore}
	variants := currinfra.Variants{Store: currStore}
	assignments := currinfra.Assignments{Store: currStore}
	blobs := currinfra.NewDiskBlobStore(t.TempDir())
	lookup := ticketsinfra.CurriculumLookup{Lessons: lessons, Variants: variants, Topics: topics}
	issuer := ticketsapp.IssueAvailable{Attempts: attemptsRepo}

	createTicketUC := ticketsapp.CreateTicket{Tickets: ticketsRepo, Curriculum: lookup}
	setReferenceUC := ticketsapp.SetReferenceAnswer{Tickets: ticketsRepo, Catalog: catalogRepo}
	grantAttemptUC := ticketsapp.GrantAttempt{Attempts: attemptsRepo, Curriculum: lookup}
	groupsAdapter := currinfra.GroupsAdapter{Groups: groupsRepo}
	ticketsHandlers := ticketspresentation.NewAPI(ticketspresentation.Deps{
		ListIncidentTypes:    ticketsapp.ListIncidentTypes{Catalog: catalogRepo},
		ListTagsByType:       ticketsapp.ListTagsByType{Catalog: catalogRepo},
		ListServices:         ticketsapp.ListServices{Catalog: catalogRepo},
		RecommendServices:    ticketsapp.RecommendServices{Catalog: catalogRepo},
		CreateTicket:         createTicketUC,
		CreateLibraryTicket:  ticketsapp.CreateLibraryTicket{Tickets: ticketsRepo, Curriculum: lookup},
		ListLibraryTickets:   ticketsapp.ListLibraryTickets{Tickets: ticketsRepo},
		ListTicketsByVariant: ticketsapp.ListTicketsByVariant{Tickets: ticketsRepo, Attempts: attemptsRepo},
		CopyTicketFromPool:   ticketsapp.CopyTicketFromPool{Tickets: ticketsRepo, Curriculum: lookup},
		GetTicket:            ticketsapp.GetTicket{Tickets: ticketsRepo, Attempts: attemptsRepo},
		UpdateTicket:         ticketsapp.UpdateTicket{Tickets: ticketsRepo, Curriculum: lookup},
		DeleteTicket:         ticketsapp.DeleteTicket{Tickets: ticketsRepo},
		SetReferenceAnswer:   setReferenceUC,
		GrantAttempt:         grantAttemptUC,
		OpenVariant: ticketsapp.OpenVariant{
			Grant: grantAttemptUC, Groups: groupsAdapter, Modules: assignments,
		},
		StartAttempt:      ticketsapp.StartAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Curriculum: lookup},
		SaveAttemptAnswer: ticketsapp.SaveAttemptAnswer{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo},
		SubmitAttempt:     ticketsapp.SubmitAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo},
		GetMyAttempt:      ticketsapp.GetMyAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo},
		ListMyAttempts:    ticketsapp.ListMyAttempts{Attempts: attemptsRepo},
		GetAttemptReport:  ticketsapp.GetAttemptReport{Attempts: attemptsRepo},
		Authenticate:      ticketsSessionAuthenticator{auth: authenticate},
	})

	currHandlers := currpresentation.NewAPI(currpresentation.Deps{
		CreateTopic:      currapp.CreateTopic{Topics: topics},
		ListTopics:       currapp.ListTopics{Topics: topics, Assignments: assignments},
		GetTopic:         currapp.GetTopic{Topics: topics, Assignments: assignments},
		UpdateTopic:      currapp.UpdateTopic{Topics: topics},
		DeleteTopic:      currapp.DeleteTopic{Topics: topics},
		CreateArticle:    currapp.CreateArticle{Topics: topics, Articles: articles},
		ListArticles:     currapp.ListArticles{Topics: topics, Articles: articles, Assignments: assignments},
		GetArticle:       currapp.GetArticle{Articles: articles, Assignments: assignments},
		UpdateArticle:    currapp.UpdateArticle{Articles: articles},
		DeleteArticle:    currapp.DeleteArticle{Articles: articles},
		AddAttachment:    currapp.AddAttachment{Articles: articles, Attachments: attachments, Blobs: blobs},
		ListAttachments:  currapp.ListAttachments{Articles: articles, Attachments: attachments, Assignments: assignments},
		GetAttachment:    currapp.GetAttachment{Articles: articles, Attachments: attachments, Assignments: assignments, Blobs: blobs},
		DeleteAttachment: currapp.DeleteAttachment{Attachments: attachments, Blobs: blobs},
		CreateModule:     currapp.CreateModule{Modules: modules},
		ListModules:      currapp.ListModules{Modules: modules, Assignments: assignments},
		GetModule:        currapp.GetModule{Modules: modules, Assignments: assignments},
		GetModuleSummary: currapp.GetModuleSummary{Modules: modules, Lessons: lessons, Variants: variants},
		UpdateModule:     currapp.UpdateModule{Modules: modules},
		DeleteModule:     currapp.DeleteModule{Modules: modules},
		CreateLesson:     currapp.CreateLesson{Modules: modules, Lessons: lessons},
		ListLessons:      currapp.ListLessons{Modules: modules, Lessons: lessons, Assignments: assignments},
		ListLessonsPool:  currapp.ListLessonsPool{Lessons: lessons},
		CopyLessonPool:   currapp.CopyLessonFromPool{Modules: modules, Lessons: lessons, Variants: variants},
		ArchiveLesson:    currapp.ArchiveLesson{Lessons: lessons},
		UpdateLesson:     currapp.UpdateLesson{Lessons: lessons},
		DeleteLesson:     currapp.DeleteLesson{Lessons: lessons},
		CreateVariant:    currapp.CreateVariant{Lessons: lessons, Variants: variants},
		ListVariants:     currapp.ListVariants{Lessons: lessons, Variants: variants},
		GetVariant:       currapp.GetVariant{Variants: variants},
		UpdateVariant:    currapp.UpdateVariant{Variants: variants},
		CloneVariant:     currapp.CloneVariant{Variants: variants},
		DeleteVariant:    currapp.DeleteVariant{Variants: variants},
		AssignModule: currapp.AssignModule{
			Modules: modules, Lessons: lessons, Variants: variants, Assignments: assignments,
			Groups: groupsAdapter, Attempts: issuer,
		},
		ListMyModules: currapp.ListMyModules{
			Modules: modules, Lessons: lessons, Variants: variants, Assignments: assignments,
			Attempts: currinfra.AttemptViews{Attempts: attemptsRepo},
		},
		Authenticate: curriculumSessionAuthenticator{auth: authenticate},
	})

	jobsRepo := geninfra.NewJobRepository(pool)
	genCatalog := testGenerationCatalog{tickets: catalogRepo}
	genPublisher := testGenerationPublisher{pool: pool}
	genHandlers := genpresentation.NewAPI(genpresentation.Deps{
		CreateJob: genapp.CreateJob{Jobs: jobsRepo, Curriculum: testGenerationCurriculum{lookup: lookup}},
		ListJobs:  genapp.ListJobs{Jobs: jobsRepo},
		GetJob:    genapp.GetJob{Jobs: jobsRepo},
		PatchJob:  genapp.PatchJob{Jobs: jobsRepo},
		RetryJob:  genapp.RetryJob{Jobs: jobsRepo},
		CancelJob: genapp.CancelJob{Jobs: jobsRepo},
		DeleteJob: genapp.DeleteJob{Jobs: jobsRepo},
		ApproveJob: genapp.ApproveJob{
			Jobs: jobsRepo, Catalog: genCatalog, Publisher: genPublisher,
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
	currpresentation.Register(api, currHandlers)
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

type testGenerationCurriculum struct {
	lookup ticketsinfra.CurriculumLookup
}

func (c testGenerationCurriculum) VariantExists(ctx context.Context, variantID uuid.UUID) error {
	return mapTicketsToGeneration(c.lookup.VariantExists(ctx, variantID))
}

func (c testGenerationCurriculum) TopicExists(ctx context.Context, topicID uuid.UUID) error {
	return mapTicketsToGeneration(c.lookup.TopicExists(ctx, topicID))
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
