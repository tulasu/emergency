package presentation

import (
	"context"

	"traineebox/internal/tickets/application"

	"github.com/google/uuid"
)

type LibraryTicketExtras struct {
	IncidentTypeCode string
	IncidentType     string
	SlotsTotal       int
	SlotsRequired    int
	VariantUsage     int
}

type ticketExtrasReader interface {
	LibraryExtras(ctx context.Context, ticketID uuid.UUID) (LibraryTicketExtras, error)
}

type API struct {
	listIncidentTypes    application.ListIncidentTypes
	listTagsByType       application.ListTagsByType
	listServices         application.ListServices
	recommendServices    application.RecommendServices
	createTicket         application.CreateTicket
	createLibraryTicket  application.CreateLibraryTicket
	listLibraryTickets   application.ListLibraryTickets
	listTicketsByVariant application.ListTicketsByVariant
	copyTicketFromPool   application.CopyTicketFromPool
	getTicket            application.GetTicket
	updateTicket         application.UpdateTicket
	deleteTicket         application.DeleteTicket
	getReferenceAnswer   application.GetReferenceAnswer
	setReferenceAnswer   application.SetReferenceAnswer
	grantAttempt         application.GrantAttempt
	openVariant          application.OpenVariant
	startAttempt         application.StartAttempt
	saveAttemptAnswer    application.SaveAttemptAnswer
	submitAttempt        application.SubmitAttempt
	getMyAttempt         application.GetMyAttempt
	listMyAttempts       application.ListMyAttempts
	listVariantAttempts  application.ListVariantAttempts
	getAttemptReport     application.GetAttemptReport
	authenticate         application.Authenticator
	ticketExtras         ticketExtrasReader
}

type Deps struct {
	ListIncidentTypes    application.ListIncidentTypes
	ListTagsByType       application.ListTagsByType
	ListServices         application.ListServices
	RecommendServices    application.RecommendServices
	CreateTicket         application.CreateTicket
	CreateLibraryTicket  application.CreateLibraryTicket
	ListLibraryTickets   application.ListLibraryTickets
	ListTicketsByVariant application.ListTicketsByVariant
	CopyTicketFromPool   application.CopyTicketFromPool
	GetTicket            application.GetTicket
	UpdateTicket         application.UpdateTicket
	DeleteTicket         application.DeleteTicket
	GetReferenceAnswer   application.GetReferenceAnswer
	SetReferenceAnswer   application.SetReferenceAnswer
	GrantAttempt         application.GrantAttempt
	OpenVariant          application.OpenVariant
	StartAttempt         application.StartAttempt
	SaveAttemptAnswer    application.SaveAttemptAnswer
	SubmitAttempt        application.SubmitAttempt
	GetMyAttempt         application.GetMyAttempt
	ListMyAttempts       application.ListMyAttempts
	ListVariantAttempts  application.ListVariantAttempts
	GetAttemptReport     application.GetAttemptReport
	Authenticate         application.Authenticator
	TicketExtras         ticketExtrasReader
}

func NewAPI(deps Deps) *API {
	return &API{
		listIncidentTypes:    deps.ListIncidentTypes,
		listTagsByType:       deps.ListTagsByType,
		listServices:         deps.ListServices,
		recommendServices:    deps.RecommendServices,
		createTicket:         deps.CreateTicket,
		createLibraryTicket:  deps.CreateLibraryTicket,
		listLibraryTickets:   deps.ListLibraryTickets,
		listTicketsByVariant: deps.ListTicketsByVariant,
		copyTicketFromPool:   deps.CopyTicketFromPool,
		getTicket:            deps.GetTicket,
		updateTicket:         deps.UpdateTicket,
		deleteTicket:         deps.DeleteTicket,
		getReferenceAnswer:   deps.GetReferenceAnswer,
		setReferenceAnswer:   deps.SetReferenceAnswer,
		grantAttempt:         deps.GrantAttempt,
		openVariant:          deps.OpenVariant,
		startAttempt:         deps.StartAttempt,
		saveAttemptAnswer:    deps.SaveAttemptAnswer,
		submitAttempt:        deps.SubmitAttempt,
		getMyAttempt:         deps.GetMyAttempt,
		listMyAttempts:       deps.ListMyAttempts,
		listVariantAttempts:  deps.ListVariantAttempts,
		getAttemptReport:     deps.GetAttemptReport,
		authenticate:         deps.Authenticate,
		ticketExtras:         deps.TicketExtras,
	}
}
