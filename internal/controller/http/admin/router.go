package admin

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/middleware"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
)

func NewAuthRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, auth usecase.AdminAuth) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, auth: auth}

	authPublicGroup := apiAdminGroup.Group("/auth")
	{
		authPublicGroup.Post("/login", r.login)
		authPublicGroup.Post("/reset", r.resetPassword)
		authPublicGroup.Post("/logout", r.logout)
	}

	authAuthGroup := apiAdminGroup.Group("/auth", middleware.NewAdminSessionMiddleware(store))
	{
		authAuthGroup.Get("/profile", r.getProfile)
		authAuthGroup.Put("/profile", r.updateProfile)
		authAuthGroup.Put("/password", r.changePassword)
		authAuthGroup.Post("/2fa/setup", r.twoFASetup)
		authAuthGroup.Post("/2fa/verify", r.twoFAVerify)
		authAuthGroup.Post("/2fa/disable", r.twoFADisable)
		authAuthGroup.Post("/2fa/recovery/reset", r.twoFARecoveryReset)
	}
}

func NewFileRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, file usecase.File) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, file: file}

	fileAuthGroup := apiAdminGroup.Group("/files", middleware.NewAdminSessionMiddleware(store))
	{
		fileAuthGroup.Get("/", r.listFiles)
		fileAuthGroup.Post("/upload-url", r.generateUploadURL)
		fileAuthGroup.Delete("/*", r.deleteFile)
	}
}

func NewUserRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, auth usecase.UserAuth, user usecase.User) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, userAuth: auth, user: user}

	usersAuthGroup := apiAdminGroup.Group("/users", middleware.NewAdminSessionMiddleware(store))
	{
		usersAuthGroup.Get("/", r.listUsers)
		usersAuthGroup.Put("/:id/status", r.updateUserStatus)
	}
}

func NewContentRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, content usecase.Content) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, content: content}

	contentAuthGroup := apiAdminGroup.Group("/content", middleware.NewAdminSessionMiddleware(store))
	{
		contentAuthGroup.Get("/posts", r.listPosts)
		contentAuthGroup.Get("/posts/:id", r.getPost)
		contentAuthGroup.Post("/posts", r.createPost)
		contentAuthGroup.Put("/posts/:id", r.updatePost)
		contentAuthGroup.Delete("/posts/:id", r.deletePost)
		contentAuthGroup.Get("/categories", r.listCategories)
		contentAuthGroup.Post("/categories", r.createCategory)
		contentAuthGroup.Put("/categories/:id", r.updateCategory)
		contentAuthGroup.Delete("/categories/:id", r.deleteCategory)
		contentAuthGroup.Get("/tags", r.listTags)
		contentAuthGroup.Post("/tags", r.createTag)
		contentAuthGroup.Put("/tags/:id", r.updateTag)
		contentAuthGroup.Delete("/tags/:id", r.deleteTag)
		contentAuthGroup.Post("/generate-slug", r.generateSlug)
	}
}

func NewCommentRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, comment usecase.Comment) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, comment: comment}

	commentsAuthGroup := apiAdminGroup.Group("/comments", middleware.NewAdminSessionMiddleware(store))
	{
		commentsAuthGroup.Get("/", r.listComments)
		commentsAuthGroup.Put("/:id/status", r.updateCommentStatus)
		commentsAuthGroup.Delete("/:id", r.deleteComment)
	}
}

func NewFeedbackRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, feedback usecase.Feedback) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, feedback: feedback}

	feedbacksAuthGroup := apiAdminGroup.Group("/feedbacks", middleware.NewAdminSessionMiddleware(store))
	{
		feedbacksAuthGroup.Get("/", r.listFeedbacks)
		feedbacksAuthGroup.Get("/:id", r.getFeedback)
		feedbacksAuthGroup.Put("/:id/status", r.updateFeedbackStatus)
		feedbacksAuthGroup.Delete("/:id", r.deleteFeedback)
	}
}

func NewLinkRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, link usecase.Link) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, link: link}

	linksAuthGroup := apiAdminGroup.Group("/links", middleware.NewAdminSessionMiddleware(store))
	{
		linksAuthGroup.Get("/", r.listLinks)
		linksAuthGroup.Post("/", r.createLink)
		linksAuthGroup.Put("/:id", r.updateLink)
		linksAuthGroup.Delete("/:id", r.deleteLink)
	}
}

func NewSettingRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, setting usecase.Setting, file usecase.File) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, setting: setting, file: file}

	settingsAuthGroup := apiAdminGroup.Group("/settings", middleware.NewAdminSessionMiddleware(store))
	{
		settingsAuthGroup.Get("/", r.listSettings)
		settingsAuthGroup.Get("/:key", r.getSettingByKey)
		settingsAuthGroup.Put("/:key", r.upsertSetting)
	}
}

func NewNotificationRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, notify usecase.Notification) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, notify: notify}

	notifAuthGroup := apiAdminGroup.Group("/notifications", middleware.NewAdminSessionMiddleware(store))
	{
		notifAuthGroup.Post("/", r.sendNotification)
	}
}
