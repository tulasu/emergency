package presentation

import (
	"traineebox/internal/tickets/application"
)

type API struct {
	listIncidentTypes  application.ListIncidentTypes
	listTagsByType     application.ListTagsByType
	listServices       application.ListServices
	recommendServices  application.RecommendServices
	createTicket       application.CreateTicket
	listTicketsByGroup application.ListTicketsByGroup
	getTicket          application.GetTicket
	setReferenceAnswer application.SetReferenceAnswer
	startAttempt       application.StartAttempt
	saveAttemptAnswer  application.SaveAttemptAnswer
	submitAttempt      application.SubmitAttempt
	getMyAttempt       application.GetMyAttempt
	listMyAttempts     application.ListMyAttempts
	authenticate       application.Authenticator
}

type Deps struct {
	ListIncidentTypes  application.ListIncidentTypes
	ListTagsByType     application.ListTagsByType
	ListServices       application.ListServices
	RecommendServices  application.RecommendServices
	CreateTicket       application.CreateTicket
	ListTicketsByGroup application.ListTicketsByGroup
	GetTicket          application.GetTicket
	SetReferenceAnswer application.SetReferenceAnswer
	StartAttempt       application.StartAttempt
	SaveAttemptAnswer  application.SaveAttemptAnswer
	SubmitAttempt      application.SubmitAttempt
	GetMyAttempt       application.GetMyAttempt
	ListMyAttempts     application.ListMyAttempts
	Authenticate       application.Authenticator
}

func NewAPI(deps Deps) *API {
	return &API{
		listIncidentTypes:  deps.ListIncidentTypes,
		listTagsByType:     deps.ListTagsByType,
		listServices:       deps.ListServices,
		recommendServices:  deps.RecommendServices,
		createTicket:       deps.CreateTicket,
		listTicketsByGroup: deps.ListTicketsByGroup,
		getTicket:          deps.GetTicket,
		setReferenceAnswer: deps.SetReferenceAnswer,
		startAttempt:       deps.StartAttempt,
		saveAttemptAnswer:  deps.SaveAttemptAnswer,
		submitAttempt:      deps.SubmitAttempt,
		getMyAttempt:       deps.GetMyAttempt,
		listMyAttempts:     deps.ListMyAttempts,
		authenticate:       deps.Authenticate,
	}
}
