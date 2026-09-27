package presentation

import (
	"traineebox/internal/tickets/application"
)

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
	setReferenceAnswer   application.SetReferenceAnswer
	grantAttempt         application.GrantAttempt
	openVariant          application.OpenVariant
	startAttempt         application.StartAttempt
	saveAttemptAnswer    application.SaveAttemptAnswer
	submitAttempt        application.SubmitAttempt
	getMyAttempt         application.GetMyAttempt
	listMyAttempts       application.ListMyAttempts
	getAttemptReport     application.GetAttemptReport
	authenticate         application.Authenticator
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
	SetReferenceAnswer   application.SetReferenceAnswer
	GrantAttempt         application.GrantAttempt
	OpenVariant          application.OpenVariant
	StartAttempt         application.StartAttempt
	SaveAttemptAnswer    application.SaveAttemptAnswer
	SubmitAttempt        application.SubmitAttempt
	GetMyAttempt         application.GetMyAttempt
	ListMyAttempts       application.ListMyAttempts
	GetAttemptReport     application.GetAttemptReport
	Authenticate         application.Authenticator
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
		setReferenceAnswer:   deps.SetReferenceAnswer,
		grantAttempt:         deps.GrantAttempt,
		openVariant:          deps.OpenVariant,
		startAttempt:         deps.StartAttempt,
		saveAttemptAnswer:    deps.SaveAttemptAnswer,
		submitAttempt:        deps.SubmitAttempt,
		getMyAttempt:         deps.GetMyAttempt,
		listMyAttempts:       deps.ListMyAttempts,
		getAttemptReport:     deps.GetAttemptReport,
		authenticate:         deps.Authenticate,
	}
}
