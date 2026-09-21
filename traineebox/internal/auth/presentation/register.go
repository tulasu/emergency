package presentation

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Tags:        []string{"System"},
	}, a.health)

	huma.Register(api, huma.Operation{
		OperationID: "create-user",
		Method:      http.MethodPost,
		Path:        "/auth/users",
		Summary:     "Create user",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.createUserHandler)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Login",
		Tags:        []string{"Auth"},
	}, a.loginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Logout",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.logoutHandler)

	huma.Register(api, huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/auth/me",
		Summary:     "Current user",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.meHandler)

	huma.Register(api, huma.Operation{
		OperationID: "block-user",
		Method:      http.MethodPost,
		Path:        "/auth/users/{userId}/block",
		Summary:     "Block or unblock user",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.blockUserHandler)

	huma.Register(api, huma.Operation{
		OperationID: "change-role",
		Method:      http.MethodPost,
		Path:        "/auth/users/{userId}/role",
		Summary:     "Change user role",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.changeRoleHandler)
}
