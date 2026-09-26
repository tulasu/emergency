package presentation

import "traineebox/internal/curriculum/application"

type API struct {
	createTopic      application.CreateTopic
	listTopics       application.ListTopics
	getTopic         application.GetTopic
	updateTopic      application.UpdateTopic
	deleteTopic      application.DeleteTopic
	createArticle    application.CreateArticle
	listArticles     application.ListArticles
	getArticle       application.GetArticle
	updateArticle    application.UpdateArticle
	deleteArticle    application.DeleteArticle
	addAttachment    application.AddAttachment
	listAttachments  application.ListAttachments
	getAttachment    application.GetAttachment
	deleteAttachment application.DeleteAttachment
	createModule     application.CreateModule
	listModules      application.ListModules
	getModule        application.GetModule
	updateModule     application.UpdateModule
	deleteModule     application.DeleteModule
	createLesson     application.CreateLesson
	listLessons      application.ListLessons
	updateLesson     application.UpdateLesson
	deleteLesson     application.DeleteLesson
	createVariant    application.CreateVariant
	listVariants     application.ListVariants
	getVariant       application.GetVariant
	updateVariant    application.UpdateVariant
	deleteVariant    application.DeleteVariant
	assignModule     application.AssignModule
	listMyModules    application.ListMyModules
	authenticate     application.Authenticator
}

type Deps struct {
	CreateTopic      application.CreateTopic
	ListTopics       application.ListTopics
	GetTopic         application.GetTopic
	UpdateTopic      application.UpdateTopic
	DeleteTopic      application.DeleteTopic
	CreateArticle    application.CreateArticle
	ListArticles     application.ListArticles
	GetArticle       application.GetArticle
	UpdateArticle    application.UpdateArticle
	DeleteArticle    application.DeleteArticle
	AddAttachment    application.AddAttachment
	ListAttachments  application.ListAttachments
	GetAttachment    application.GetAttachment
	DeleteAttachment application.DeleteAttachment
	CreateModule     application.CreateModule
	ListModules      application.ListModules
	GetModule        application.GetModule
	UpdateModule     application.UpdateModule
	DeleteModule     application.DeleteModule
	CreateLesson     application.CreateLesson
	ListLessons      application.ListLessons
	UpdateLesson     application.UpdateLesson
	DeleteLesson     application.DeleteLesson
	CreateVariant    application.CreateVariant
	ListVariants     application.ListVariants
	GetVariant       application.GetVariant
	UpdateVariant    application.UpdateVariant
	DeleteVariant    application.DeleteVariant
	AssignModule     application.AssignModule
	ListMyModules    application.ListMyModules
	Authenticate     application.Authenticator
}

func NewAPI(deps Deps) *API {
	return &API{
		createTopic:      deps.CreateTopic,
		listTopics:       deps.ListTopics,
		getTopic:         deps.GetTopic,
		updateTopic:      deps.UpdateTopic,
		deleteTopic:      deps.DeleteTopic,
		createArticle:    deps.CreateArticle,
		listArticles:     deps.ListArticles,
		getArticle:       deps.GetArticle,
		updateArticle:    deps.UpdateArticle,
		deleteArticle:    deps.DeleteArticle,
		addAttachment:    deps.AddAttachment,
		listAttachments:  deps.ListAttachments,
		getAttachment:    deps.GetAttachment,
		deleteAttachment: deps.DeleteAttachment,
		createModule:     deps.CreateModule,
		listModules:      deps.ListModules,
		getModule:        deps.GetModule,
		updateModule:     deps.UpdateModule,
		deleteModule:     deps.DeleteModule,
		createLesson:     deps.CreateLesson,
		listLessons:      deps.ListLessons,
		updateLesson:     deps.UpdateLesson,
		deleteLesson:     deps.DeleteLesson,
		createVariant:    deps.CreateVariant,
		listVariants:     deps.ListVariants,
		getVariant:       deps.GetVariant,
		updateVariant:    deps.UpdateVariant,
		deleteVariant:    deps.DeleteVariant,
		assignModule:     deps.AssignModule,
		listMyModules:    deps.ListMyModules,
		authenticate:     deps.Authenticate,
	}
}
