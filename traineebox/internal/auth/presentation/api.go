package presentation

import (
	"traineebox/internal/auth/application"
)

type API struct {
	version        string
	createUser     application.CreateUser
	login          application.Login
	logout         application.Logout
	me             application.Me
	blockUser      application.BlockUser
	changeRole     application.ChangeRole
	listUsers      application.ListUsers
	getUser        application.GetUser
	changePassword application.ChangePassword
	authenticate   application.Authenticate
	provision      application.ProvisionUsers
}

type Deps struct {
	Version        string
	CreateUser     application.CreateUser
	Login          application.Login
	Logout         application.Logout
	Me             application.Me
	BlockUser      application.BlockUser
	ChangeRole     application.ChangeRole
	ListUsers      application.ListUsers
	GetUser        application.GetUser
	ChangePassword application.ChangePassword
	Authenticate   application.Authenticate
	Provision      application.ProvisionUsers
}

func NewAPI(deps Deps) *API {
	return &API{
		version:        deps.Version,
		createUser:     deps.CreateUser,
		login:          deps.Login,
		logout:         deps.Logout,
		me:             deps.Me,
		blockUser:      deps.BlockUser,
		changeRole:     deps.ChangeRole,
		listUsers:      deps.ListUsers,
		getUser:        deps.GetUser,
		changePassword: deps.ChangePassword,
		authenticate:   deps.Authenticate,
		provision:      deps.Provision,
	}
}
