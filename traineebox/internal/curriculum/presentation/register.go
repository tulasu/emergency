package presentation

import (
	"context"
	"io"
	"net/http"

	"traineebox/internal/curriculum/application"
	"traineebox/internal/curriculum/domain/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	sec := []map[string][]string{{"session": {}}}

	huma.Register(api, huma.Operation{OperationID: "create-topic", Method: http.MethodPost, Path: "/topics", Summary: "Create topic", Tags: []string{"Topics"}, Security: sec}, a.createTopicHandler)
	huma.Register(api, huma.Operation{OperationID: "list-topics", Method: http.MethodGet, Path: "/topics", Summary: "List topics", Tags: []string{"Topics"}, Security: sec}, a.listTopicsHandler)
	huma.Register(api, huma.Operation{OperationID: "get-topic", Method: http.MethodGet, Path: "/topics/{topicId}", Summary: "Get topic", Tags: []string{"Topics"}, Security: sec}, a.getTopicHandler)
	huma.Register(api, huma.Operation{OperationID: "update-topic", Method: http.MethodPatch, Path: "/topics/{topicId}", Summary: "Update topic", Tags: []string{"Topics"}, Security: sec}, a.updateTopicHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-topic", Method: http.MethodDelete, Path: "/topics/{topicId}", Summary: "Delete topic", Tags: []string{"Topics"}, Security: sec}, a.deleteTopicHandler)

	huma.Register(api, huma.Operation{OperationID: "create-article", Method: http.MethodPost, Path: "/topics/{topicId}/articles", Summary: "Create article", Tags: []string{"Articles"}, Security: sec}, a.createArticleHandler)
	huma.Register(api, huma.Operation{OperationID: "list-articles", Method: http.MethodGet, Path: "/topics/{topicId}/articles", Summary: "List articles", Tags: []string{"Articles"}, Security: sec}, a.listArticlesHandler)
	huma.Register(api, huma.Operation{OperationID: "get-article", Method: http.MethodGet, Path: "/articles/{articleId}", Summary: "Get article", Tags: []string{"Articles"}, Security: sec}, a.getArticleHandler)
	huma.Register(api, huma.Operation{OperationID: "update-article", Method: http.MethodPatch, Path: "/articles/{articleId}", Summary: "Update article", Tags: []string{"Articles"}, Security: sec}, a.updateArticleHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-article", Method: http.MethodDelete, Path: "/articles/{articleId}", Summary: "Delete article", Tags: []string{"Articles"}, Security: sec}, a.deleteArticleHandler)

	huma.Register(api, huma.Operation{OperationID: "add-attachment", Method: http.MethodPost, Path: "/articles/{articleId}/attachments", Summary: "Upload article attachment", Tags: []string{"Articles"}, Security: sec}, a.addAttachmentHandler)
	huma.Register(api, huma.Operation{OperationID: "list-attachments", Method: http.MethodGet, Path: "/articles/{articleId}/attachments", Summary: "List attachments", Tags: []string{"Articles"}, Security: sec}, a.listAttachmentsHandler)
	huma.Register(api, huma.Operation{OperationID: "get-attachment", Method: http.MethodGet, Path: "/attachments/{attachmentId}", Summary: "Download attachment", Tags: []string{"Articles"}, Security: sec}, a.getAttachmentHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-attachment", Method: http.MethodDelete, Path: "/attachments/{attachmentId}", Summary: "Delete attachment", Tags: []string{"Articles"}, Security: sec}, a.deleteAttachmentHandler)

	huma.Register(api, huma.Operation{OperationID: "create-module", Method: http.MethodPost, Path: "/modules", Summary: "Create module", Tags: []string{"Modules"}, Security: sec}, a.createModuleHandler)
	huma.Register(api, huma.Operation{OperationID: "list-modules", Method: http.MethodGet, Path: "/modules", Summary: "List modules", Tags: []string{"Modules"}, Security: sec}, a.listModulesHandler)
	huma.Register(api, huma.Operation{OperationID: "get-module", Method: http.MethodGet, Path: "/modules/{moduleId}", Summary: "Get module", Tags: []string{"Modules"}, Security: sec}, a.getModuleHandler)
	huma.Register(api, huma.Operation{OperationID: "update-module", Method: http.MethodPatch, Path: "/modules/{moduleId}", Summary: "Update module", Tags: []string{"Modules"}, Security: sec}, a.updateModuleHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-module", Method: http.MethodDelete, Path: "/modules/{moduleId}", Summary: "Delete module", Tags: []string{"Modules"}, Security: sec}, a.deleteModuleHandler)
	huma.Register(api, huma.Operation{OperationID: "list-my-modules", Method: http.MethodGet, Path: "/me/modules", Summary: "List my assigned modules", Tags: []string{"Modules"}, Security: sec}, a.listMyModulesHandler)

	huma.Register(api, huma.Operation{OperationID: "create-lesson", Method: http.MethodPost, Path: "/modules/{moduleId}/lessons", Summary: "Create lesson", Tags: []string{"Lessons"}, Security: sec}, a.createLessonHandler)
	huma.Register(api, huma.Operation{OperationID: "list-lessons", Method: http.MethodGet, Path: "/modules/{moduleId}/lessons", Summary: "List lessons", Tags: []string{"Lessons"}, Security: sec}, a.listLessonsHandler)
	huma.Register(api, huma.Operation{OperationID: "update-lesson", Method: http.MethodPatch, Path: "/lessons/{lessonId}", Summary: "Update lesson", Tags: []string{"Lessons"}, Security: sec}, a.updateLessonHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-lesson", Method: http.MethodDelete, Path: "/lessons/{lessonId}", Summary: "Delete lesson", Tags: []string{"Lessons"}, Security: sec}, a.deleteLessonHandler)

	huma.Register(api, huma.Operation{OperationID: "create-variant", Method: http.MethodPost, Path: "/lessons/{lessonId}/variants", Summary: "Create variant", Tags: []string{"Variants"}, Security: sec}, a.createVariantHandler)
	huma.Register(api, huma.Operation{OperationID: "list-variants", Method: http.MethodGet, Path: "/lessons/{lessonId}/variants", Summary: "List variants", Tags: []string{"Variants"}, Security: sec}, a.listVariantsHandler)
	huma.Register(api, huma.Operation{OperationID: "get-variant", Method: http.MethodGet, Path: "/variants/{variantId}", Summary: "Get variant", Tags: []string{"Variants"}, Security: sec}, a.getVariantHandler)
	huma.Register(api, huma.Operation{OperationID: "update-variant", Method: http.MethodPatch, Path: "/variants/{variantId}", Summary: "Update variant", Tags: []string{"Variants"}, Security: sec}, a.updateVariantHandler)
	huma.Register(api, huma.Operation{OperationID: "delete-variant", Method: http.MethodDelete, Path: "/variants/{variantId}", Summary: "Delete variant", Tags: []string{"Variants"}, Security: sec}, a.deleteVariantHandler)

	huma.Register(api, huma.Operation{OperationID: "assign-module-group", Method: http.MethodPost, Path: "/groups/{groupId}/modules", Summary: "Assign module to group students", Tags: []string{"Assignments"}, Security: sec}, a.assignGroupHandler)
	huma.Register(api, huma.Operation{OperationID: "assign-module-user", Method: http.MethodPost, Path: "/users/{userId}/modules", Summary: "Assign module to user", Tags: []string{"Assignments"}, Security: sec}, a.assignUserHandler)
}

func (a *API) createTopicHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		Title string `json:"title" minLength:"1" maxLength:"256"`
	}
}) (*struct{ Body topicDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	topic, err := a.createTopic.Execute(ctx, application.CreateTopicInput{ActorID: user.ID, Role: user.Role, Title: in.Body.Title})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body topicDTO }{Body: toTopicDTO(topic)}, nil
}

func (a *API) listTopicsHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
}) (*struct{ Body []topicDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listTopics.Execute(ctx, application.ListTopicsInput{ActorID: user.ID, Role: user.Role})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]topicDTO, 0, len(items))
	for _, t := range items {
		out = append(out, toTopicDTO(t))
	}
	return &struct{ Body []topicDTO }{Body: out}, nil
}

func (a *API) getTopicHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TopicID       uuid.UUID `path:"topicId"`
}) (*struct{ Body topicDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	topic, err := a.getTopic.Execute(ctx, application.GetTopicInput{ActorID: user.ID, Role: user.Role, TopicID: in.TopicID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body topicDTO }{Body: toTopicDTO(topic)}, nil
}

func (a *API) updateTopicHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TopicID       uuid.UUID `path:"topicId"`
	Body          struct {
		Title string `json:"title" minLength:"1" maxLength:"256"`
	}
}) (*struct{ Body topicDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	topic, err := a.updateTopic.Execute(ctx, application.UpdateTopicInput{Role: user.Role, TopicID: in.TopicID, Title: in.Body.Title})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body topicDTO }{Body: toTopicDTO(topic)}, nil
}

func (a *API) deleteTopicHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TopicID       uuid.UUID `path:"topicId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteTopic.Execute(ctx, user.Role, in.TopicID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) createArticleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TopicID       uuid.UUID `path:"topicId"`
	Body          struct {
		Title  string `json:"title" minLength:"1" maxLength:"256"`
		BodyMD string `json:"body_md"`
	}
}) (*struct{ Body articleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	article, err := a.createArticle.Execute(ctx, application.CreateArticleInput{
		ActorID: user.ID, Role: user.Role, TopicID: in.TopicID, Title: in.Body.Title, BodyMD: in.Body.BodyMD,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body articleDTO }{Body: toArticleDTO(article)}, nil
}

func (a *API) listArticlesHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	TopicID       uuid.UUID `path:"topicId"`
}) (*struct{ Body []articleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listArticles.Execute(ctx, application.ListArticlesInput{ActorID: user.ID, Role: user.Role, TopicID: in.TopicID})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]articleDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toArticleDTO(it))
	}
	return &struct{ Body []articleDTO }{Body: out}, nil
}

func (a *API) getArticleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ArticleID     uuid.UUID `path:"articleId"`
}) (*struct{ Body articleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	article, err := a.getArticle.Execute(ctx, application.GetArticleInput{ActorID: user.ID, Role: user.Role, ArticleID: in.ArticleID})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body articleDTO }{Body: toArticleDTO(article)}, nil
}

func (a *API) updateArticleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ArticleID     uuid.UUID `path:"articleId"`
	Body          struct {
		Title  string `json:"title" minLength:"1" maxLength:"256"`
		BodyMD string `json:"body_md"`
	}
}) (*struct{ Body articleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	article, err := a.updateArticle.Execute(ctx, application.UpdateArticleInput{
		Role: user.Role, ArticleID: in.ArticleID, Title: in.Body.Title, BodyMD: in.Body.BodyMD,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body articleDTO }{Body: toArticleDTO(article)}, nil
}

func (a *API) deleteArticleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ArticleID     uuid.UUID `path:"articleId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteArticle.Execute(ctx, user.Role, in.ArticleID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) addAttachmentHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ArticleID     uuid.UUID `path:"articleId"`
	RawBody       huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" required:"true"`
	}]
}) (*struct{ Body attachmentDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	form := in.RawBody.Data()
	file := form.File
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, mapError(err)
	}
	att, err := a.addAttachment.Execute(ctx, application.AddAttachmentInput{
		Role: user.Role, ArticleID: in.ArticleID, Filename: file.Filename, ContentType: file.ContentType, Data: data,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body attachmentDTO }{Body: toAttachmentDTO(att)}, nil
}

func (a *API) listAttachmentsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ArticleID     uuid.UUID `path:"articleId"`
}) (*struct{ Body []attachmentDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listAttachments.Execute(ctx, application.ListAttachmentsInput{ActorID: user.ID, Role: user.Role, ArticleID: in.ArticleID})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]attachmentDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toAttachmentDTO(it))
	}
	return &struct{ Body []attachmentDTO }{Body: out}, nil
}

func (a *API) getAttachmentHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttachmentID  uuid.UUID `path:"attachmentId"`
}) (*huma.StreamResponse, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	att, data, err := a.getAttachment.Execute(ctx, application.GetAttachmentInput{ActorID: user.ID, Role: user.Role, AttachmentID: in.AttachmentID})
	if err != nil {
		return nil, mapError(err)
	}
	return &huma.StreamResponse{Body: func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", att.ContentType)
		ctx.SetHeader("Content-Disposition", `attachment; filename="`+att.Filename+`"`)
		ctx.SetStatus(http.StatusOK)
		_, _ = ctx.BodyWriter().Write(data)
	}}, nil
}

func (a *API) deleteAttachmentHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	AttachmentID  uuid.UUID `path:"attachmentId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteAttachment.Execute(ctx, user.Role, in.AttachmentID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) createModuleHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		Title       string `json:"title" minLength:"1" maxLength:"256"`
		Description string `json:"description"`
	}
}) (*struct{ Body moduleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	module, err := a.createModule.Execute(ctx, application.CreateModuleInput{
		ActorID: user.ID, Role: user.Role, Title: in.Body.Title, Description: in.Body.Description,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body moduleDTO }{Body: toModuleDTO(module)}, nil
}

func (a *API) listModulesHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
}) (*struct{ Body []moduleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listModules.Execute(ctx, user.ID, user.Role)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]moduleDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toModuleDTO(it))
	}
	return &struct{ Body []moduleDTO }{Body: out}, nil
}

func (a *API) getModuleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ModuleID      uuid.UUID `path:"moduleId"`
}) (*struct{ Body moduleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	module, err := a.getModule.Execute(ctx, user.ID, user.Role, in.ModuleID)
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body moduleDTO }{Body: toModuleDTO(module)}, nil
}

func (a *API) updateModuleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ModuleID      uuid.UUID `path:"moduleId"`
	Body          struct {
		Title       string `json:"title" minLength:"1" maxLength:"256"`
		Description string `json:"description"`
	}
}) (*struct{ Body moduleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	module, err := a.updateModule.Execute(ctx, application.UpdateModuleInput{
		Role: user.Role, ModuleID: in.ModuleID, Title: in.Body.Title, Description: in.Body.Description,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body moduleDTO }{Body: toModuleDTO(module)}, nil
}

func (a *API) deleteModuleHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ModuleID      uuid.UUID `path:"moduleId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteModule.Execute(ctx, user.Role, in.ModuleID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) listMyModulesHandler(ctx context.Context, in *struct {
	Authorization string `header:"Authorization"`
}) (*struct{ Body []assignedModuleDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listMyModules.Execute(ctx, user.ID, user.Role)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]assignedModuleDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toAssignedModuleDTO(it))
	}
	return &struct{ Body []assignedModuleDTO }{Body: out}, nil
}

func (a *API) createLessonHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ModuleID      uuid.UUID `path:"moduleId"`
	Body          struct {
		Title           string `json:"title" minLength:"1" maxLength:"256"`
		Position        int    `json:"position"`
		DurationSeconds *int   `json:"duration_seconds,omitempty"`
	}
}) (*struct{ Body lessonDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	lesson, err := a.createLesson.Execute(ctx, application.CreateLessonInput{
		Role: user.Role, ModuleID: in.ModuleID, Title: in.Body.Title, Position: in.Body.Position, DurationSeconds: in.Body.DurationSeconds,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body lessonDTO }{Body: toLessonDTO(lesson)}, nil
}

func (a *API) listLessonsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	ModuleID      uuid.UUID `path:"moduleId"`
}) (*struct{ Body []lessonDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listLessons.Execute(ctx, user.ID, user.Role, in.ModuleID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]lessonDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toLessonDTO(it))
	}
	return &struct{ Body []lessonDTO }{Body: out}, nil
}

func (a *API) updateLessonHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	LessonID      uuid.UUID `path:"lessonId"`
	Body          struct {
		Title           string `json:"title" minLength:"1" maxLength:"256"`
		Position        int    `json:"position"`
		DurationSeconds *int   `json:"duration_seconds,omitempty"`
	}
}) (*struct{ Body lessonDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	lesson, err := a.updateLesson.Execute(ctx, application.UpdateLessonInput{
		Role: user.Role, LessonID: in.LessonID, Title: in.Body.Title, Position: in.Body.Position, DurationSeconds: in.Body.DurationSeconds,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body lessonDTO }{Body: toLessonDTO(lesson)}, nil
}

func (a *API) deleteLessonHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	LessonID      uuid.UUID `path:"lessonId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteLesson.Execute(ctx, user.Role, in.LessonID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) createVariantHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	LessonID      uuid.UUID `path:"lessonId"`
	Body          struct {
		Title    string `json:"title" minLength:"1" maxLength:"256"`
		Position int    `json:"position"`
	}
}) (*struct{ Body variantDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	variant, err := a.createVariant.Execute(ctx, application.CreateVariantInput{
		Role: user.Role, LessonID: in.LessonID, Title: in.Body.Title, Position: in.Body.Position,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body variantDTO }{Body: toVariantDTO(variant)}, nil
}

func (a *API) listVariantsHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	LessonID      uuid.UUID `path:"lessonId"`
}) (*struct{ Body []variantDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	items, err := a.listVariants.Execute(ctx, user.ID, user.Role, in.LessonID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]variantDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toVariantDTO(it))
	}
	return &struct{ Body []variantDTO }{Body: out}, nil
}

func (a *API) getVariantHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
}) (*struct{ Body variantDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	variant, err := a.getVariant.Execute(ctx, user.Role, in.VariantID)
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body variantDTO }{Body: toVariantDTO(variant)}, nil
}

func (a *API) updateVariantHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
	Body          struct {
		Title    string `json:"title" minLength:"1" maxLength:"256"`
		Position int    `json:"position"`
	}
}) (*struct{ Body variantDTO }, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	variant, err := a.updateVariant.Execute(ctx, application.UpdateVariantInput{
		Role: user.Role, VariantID: in.VariantID, Title: in.Body.Title, Position: in.Body.Position,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &struct{ Body variantDTO }{Body: toVariantDTO(variant)}, nil
}

func (a *API) deleteVariantHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	VariantID     uuid.UUID `path:"variantId"`
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	if err := a.deleteVariant.Execute(ctx, user.Role, in.VariantID); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

type assignBody struct {
	ModuleID uuid.UUID `json:"module_id" format:"uuid"`
	Lessons  []struct {
		LessonID  uuid.UUID `json:"lesson_id" format:"uuid"`
		VariantID uuid.UUID `json:"variant_id" format:"uuid"`
	} `json:"lessons"`
}

func picksFrom(body assignBody) []models.LessonVariantPick {
	out := make([]models.LessonVariantPick, 0, len(body.Lessons))
	for _, p := range body.Lessons {
		out = append(out, models.LessonVariantPick{LessonID: p.LessonID, VariantID: p.VariantID})
	}
	return out
}

func (a *API) assignGroupHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	GroupID       uuid.UUID `path:"groupId"`
	Body          assignBody
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	gid := in.GroupID
	if err := a.assignModule.Execute(ctx, application.AssignModuleInput{
		ActorID: user.ID, Role: user.Role, GroupID: &gid, ModuleID: in.Body.ModuleID, Picks: picksFrom(in.Body),
	}); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}

func (a *API) assignUserHandler(ctx context.Context, in *struct {
	Authorization string    `header:"Authorization"`
	UserID        uuid.UUID `path:"userId"`
	Body          assignBody
}) (*struct{}, error) {
	user, err := a.requireSignedIn(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	uid := in.UserID
	if err := a.assignModule.Execute(ctx, application.AssignModuleInput{
		ActorID: user.ID, Role: user.Role, UserID: &uid, ModuleID: in.Body.ModuleID, Picks: picksFrom(in.Body),
	}); err != nil {
		return nil, mapError(err)
	}
	return &struct{}{}, nil
}
