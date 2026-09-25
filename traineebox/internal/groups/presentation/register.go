package presentation

import (
	"context"
	"net/http"

	"traineebox/internal/groups/application"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-group",
		Method:      http.MethodPost,
		Path:        "/groups",
		Summary:     "Create group",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.createGroupHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/groups",
		Summary:     "List groups",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.listGroupsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        "/groups/{groupId}",
		Summary:     "Get group",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.getGroupHandler)

	huma.Register(api, huma.Operation{
		OperationID: "rename-group",
		Method:      http.MethodPatch,
		Path:        "/groups/{groupId}",
		Summary:     "Rename group",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.renameGroupHandler)

	huma.Register(api, huma.Operation{
		OperationID: "delete-group",
		Method:      http.MethodDelete,
		Path:        "/groups/{groupId}",
		Summary:     "Delete group",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.deleteGroupHandler)

	huma.Register(api, huma.Operation{
		OperationID: "add-group-member",
		Method:      http.MethodPost,
		Path:        "/groups/{groupId}/members",
		Summary:     "Add group member",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.addMemberHandler)

	huma.Register(api, huma.Operation{
		OperationID: "remove-group-member",
		Method:      http.MethodDelete,
		Path:        "/groups/{groupId}/members/{userId}",
		Summary:     "Remove group member",
		Tags:        []string{"Groups"},
		Security:    []map[string][]string{{"session": {}}},
	}, a.removeMemberHandler)
}

type createGroupInput struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		Name string `json:"name" minLength:"1" maxLength:"128"`
	}
}

type groupOutput struct {
	Body groupDTO
}

type groupsOutput struct {
	Body []groupDTO
}

func (a *API) createGroupHandler(ctx context.Context, in *createGroupInput) (*groupOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if user.Role != value_objects.AccountRoleTeacher {
		return nil, huma.Error403Forbidden("forbidden")
	}
	group, err := a.createGroup.Execute(ctx, application.CreateGroupInput{
		ActorID: user.ID,
		Name:    in.Body.Name,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &groupOutput{Body: toGroupDTO(group)}, nil
}

func (a *API) listGroupsHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
}) (*groupsOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	groups, err := a.listGroups.Execute(ctx, application.ListGroupsInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]groupDTO, 0, len(groups))
	for _, g := range groups {
		out = append(out, toGroupDTO(g))
	}
	return &groupsOutput{Body: out}, nil
}

type groupIDInput struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
}

func (a *API) getGroupHandler(ctx context.Context, in *groupIDInput) (*groupOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	group, err := a.getGroup.Execute(ctx, application.GetGroupInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
		GroupID: in.GroupID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &groupOutput{Body: toGroupDTO(group)}, nil
}

type renameGroupInput struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	Body          struct {
		Name string `json:"name" minLength:"1" maxLength:"128"`
	}
}

func (a *API) renameGroupHandler(ctx context.Context, in *renameGroupInput) (*groupOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	group, err := a.renameGroup.Execute(ctx, application.RenameGroupInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
		GroupID: in.GroupID,
		Name:    in.Body.Name,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &groupOutput{Body: toGroupDTO(group)}, nil
}

func (a *API) deleteGroupHandler(ctx context.Context, in *groupIDInput) (*emptyOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteGroup.Execute(ctx, application.DeleteGroupInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
		GroupID: in.GroupID,
	}); err != nil {
		return nil, mapError(err)
	}
	return &emptyOutput{}, nil
}

type addMemberInput struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	Body          struct {
		UserID string `json:"user_id" format:"uuid"`
		Role   string `json:"role" enum:"teacher,student"`
	}
}

func (a *API) addMemberHandler(ctx context.Context, in *addMemberInput) (*groupOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(in.Body.UserID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid_input")
	}
	group, err := a.addMember.Execute(ctx, application.AddMemberInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
		GroupID: in.GroupID,
		UserID:  userID,
		Role:    in.Body.Role,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &groupOutput{Body: toGroupDTO(group)}, nil
}

type removeMemberInput struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	UserID        uuid.UUID `path:"userId"`
}

func (a *API) removeMemberHandler(ctx context.Context, in *removeMemberInput) (*groupOutput, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	group, err := a.removeMember.Execute(ctx, application.RemoveMemberInput{
		ActorID: user.ID,
		Admin:   isAdmin(user),
		GroupID: in.GroupID,
		UserID:  in.UserID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &groupOutput{Body: toGroupDTO(group)}, nil
}
