package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"traineebox/internal/auth/application"
	authinfra "traineebox/internal/auth/infrastructure"
	authpresentation "traineebox/internal/auth/presentation"
	currapp "traineebox/internal/curriculum/application"
	currinfra "traineebox/internal/curriculum/infrastructure"
	currpresentation "traineebox/internal/curriculum/presentation"
	genapp "traineebox/internal/generation/application"
	geninfra "traineebox/internal/generation/infrastructure"
	genpresentation "traineebox/internal/generation/presentation"
	groupsapp "traineebox/internal/groups/application"
	groupsinfra "traineebox/internal/groups/infrastructure"
	groupspresentation "traineebox/internal/groups/presentation"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"
	ticketsapp "traineebox/internal/tickets/application"
	ticketsinfra "traineebox/internal/tickets/infrastructure"
	ticketspresentation "traineebox/internal/tickets/presentation"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	users := authinfra.NewUserRepository(pool)
	sessions := authinfra.NewSessionRepository(pool)
	hasher := application.PasswordHasher{}
	authenticate := application.Authenticate{Users: users, Sessions: sessions}

	groupsRepo := groupsinfra.NewGroupRepository(pool)
	directory := groupsinfra.NewUserDirectory(pool)
	addMember := groupsapp.AddMember{Groups: groupsRepo, Directory: directory}

	authHandlers := authpresentation.NewAPI(authpresentation.Deps{
		Version:      version,
		CreateUser:   application.CreateUser{Users: users, Hasher: hasher},
		Login:        application.Login{Users: users, Sessions: sessions, Hasher: hasher, SessionTTL: cfg.SessionTTL},
		Logout:       application.Logout{Sessions: sessions},
		Me:           application.Me{Authenticate: authenticate},
		BlockUser:    application.BlockUser{Users: users},
		ChangeRole:   application.ChangeRole{Users: users},
		Authenticate: authenticate,
		Provision: application.ProvisionUsers{
			Users:  users,
			Hasher: hasher,
			Enroll: studentEnroller{add: addMember},
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

	cat, err := ticketsinfra.LoadCatalog(cfg.CatalogPath)
	if err != nil {
		log.Fatalf("catalog: %v", err)
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
	blobs := currinfra.NewDiskBlobStore(cfg.MediaPath)
	lookup := ticketsinfra.CurriculumLookup{Lessons: lessons, Variants: variants, Topics: topics}
	issuer := ticketsapp.IssueAvailable{Attempts: attemptsRepo}

	createTicketUC := ticketsapp.CreateTicket{Tickets: ticketsRepo, Curriculum: lookup}
	setReferenceUC := ticketsapp.SetReferenceAnswer{Tickets: ticketsRepo, Catalog: catalogRepo}
	ticketsHandlers := ticketspresentation.NewAPI(ticketspresentation.Deps{
		ListIncidentTypes:    ticketsapp.ListIncidentTypes{Catalog: catalogRepo},
		ListTagsByType:       ticketsapp.ListTagsByType{Catalog: catalogRepo},
		ListServices:         ticketsapp.ListServices{Catalog: catalogRepo},
		RecommendServices:    ticketsapp.RecommendServices{Catalog: catalogRepo},
		CreateTicket:         createTicketUC,
		ListTicketsByVariant: ticketsapp.ListTicketsByVariant{Tickets: ticketsRepo, Attempts: attemptsRepo},
		GetTicket:            ticketsapp.GetTicket{Tickets: ticketsRepo, Attempts: attemptsRepo},
		UpdateTicket:         ticketsapp.UpdateTicket{Tickets: ticketsRepo, Curriculum: lookup},
		DeleteTicket:         ticketsapp.DeleteTicket{Tickets: ticketsRepo},
		SetReferenceAnswer:   setReferenceUC,
		GrantAttempt:         ticketsapp.GrantAttempt{Attempts: attemptsRepo, Curriculum: lookup},
		StartAttempt:         ticketsapp.StartAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Curriculum: lookup},
		SaveAttemptAnswer:    ticketsapp.SaveAttemptAnswer{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo},
		SubmitAttempt:        ticketsapp.SubmitAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo},
		GetMyAttempt:         ticketsapp.GetMyAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo},
		ListMyAttempts:       ticketsapp.ListMyAttempts{Attempts: attemptsRepo},
		GetAttemptReport:     ticketsapp.GetAttemptReport{Attempts: attemptsRepo},
		Authenticate:         ticketsSessionAuthenticator{auth: authenticate},
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
		UpdateModule:     currapp.UpdateModule{Modules: modules},
		DeleteModule:     currapp.DeleteModule{Modules: modules},
		CreateLesson:     currapp.CreateLesson{Modules: modules, Lessons: lessons},
		ListLessons:      currapp.ListLessons{Modules: modules, Lessons: lessons, Assignments: assignments},
		UpdateLesson:     currapp.UpdateLesson{Lessons: lessons},
		DeleteLesson:     currapp.DeleteLesson{Lessons: lessons},
		CreateVariant:    currapp.CreateVariant{Lessons: lessons, Variants: variants},
		ListVariants:     currapp.ListVariants{Lessons: lessons, Variants: variants},
		GetVariant:       currapp.GetVariant{Variants: variants},
		UpdateVariant:    currapp.UpdateVariant{Variants: variants},
		DeleteVariant:    currapp.DeleteVariant{Variants: variants},
		AssignModule: currapp.AssignModule{
			Modules: modules, Lessons: lessons, Variants: variants, Assignments: assignments,
			Groups: currinfra.GroupsAdapter{Groups: groupsRepo}, Attempts: issuer,
		},
		ListMyModules: currapp.ListMyModules{
			Modules: modules, Lessons: lessons, Variants: variants, Assignments: assignments,
			Attempts: currinfra.AttemptViews{Attempts: attemptsRepo},
		},
		Authenticate: curriculumSessionAuthenticator{auth: authenticate},
	})

	jobsRepo := geninfra.NewJobRepository(pool)
	genCatalog := generationCatalog{tickets: catalogRepo}
	genPublisher := generationPublisher{pool: pool}
	genHandlers := genpresentation.NewAPI(genpresentation.Deps{
		CreateJob: genapp.CreateJob{Jobs: jobsRepo, Curriculum: generationCurriculum{inner: lookup}},
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
	apiCfg := huma.DefaultConfig("TraineeBox API", version)
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
	wireCallsAndDialog(api, pool, authenticate, ticketsRepo, attemptsRepo, jobsRepo, genCatalog)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s version=%s", cfg.HTTPAddr, version)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
