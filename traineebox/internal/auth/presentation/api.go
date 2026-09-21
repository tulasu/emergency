package presentation

import (
	"traineebox/internal/auth/application"
)

type API struct {
	version      string
	createUser   application.CreateUser
	login        application.Login
	logout       application.Logout
	me           application.Me
	blockUser    application.BlockUser
	changeRole   application.ChangeRole
	authenticate application.Authenticate
}

type Deps struct {
	Version      string
	CreateUser   application.CreateUser
	Login        application.Login
	Logout       application.Logout
	Me           application.Me
	BlockUser    application.BlockUser
	ChangeRole   application.ChangeRole
	Authenticate application.Authenticate
}

func NewAPI(deps Deps) *API {
	return &API{
		version:      deps.Version,
		createUser:   deps.CreateUser,
		login:        deps.Login,
		logout:       deps.Logout,
		me:           deps.Me,
		blockUser:    deps.BlockUser,
		changeRole:   deps.ChangeRole,
		authenticate: deps.Authenticate,
	}
}
