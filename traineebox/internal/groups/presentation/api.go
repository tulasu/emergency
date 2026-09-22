package presentation

import (
	"traineebox/internal/groups/application"
)

type API struct {
	createGroup  application.CreateGroup
	renameGroup  application.RenameGroup
	deleteGroup  application.DeleteGroup
	listGroups   application.ListGroups
	getGroup     application.GetGroup
	addMember    application.AddMember
	removeMember application.RemoveMember
	authenticate application.Authenticator
}

type Deps struct {
	CreateGroup  application.CreateGroup
	RenameGroup  application.RenameGroup
	DeleteGroup  application.DeleteGroup
	ListGroups   application.ListGroups
	GetGroup     application.GetGroup
	AddMember    application.AddMember
	RemoveMember application.RemoveMember
	Authenticate application.Authenticator
}

func NewAPI(deps Deps) *API {
	return &API{
		createGroup:  deps.CreateGroup,
		renameGroup:  deps.RenameGroup,
		deleteGroup:  deps.DeleteGroup,
		listGroups:   deps.ListGroups,
		getGroup:     deps.GetGroup,
		addMember:    deps.AddMember,
		removeMember: deps.RemoveMember,
		authenticate: deps.Authenticate,
	}
}
