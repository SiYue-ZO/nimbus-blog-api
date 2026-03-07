package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/middleware"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
)

func NewCaptchaRoutes(apiV1Group fiber.Router, l logger.Interface, c usecase.Captcha) {
	r := &V1{captcha: c, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	captchaPublicGroup := apiV1Group.Group("/captcha")
	{
		captchaPublicGroup.Get("/generate", r.generateCaptcha)
	}
}

func NewEmailRoutes(apiV1Group fiber.Router, l logger.Interface, c usecase.Captcha, e usecase.Email) {
	r := &V1{email: e, captcha: c, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	emailPublicGroup := apiV1Group.Group("/email")
	{
		emailPublicGroup.Post("/send-code", r.sendEmailCode)
	}
}

func NewAuthRoutes(apiV1Group fiber.Router, l logger.Interface, c usecase.Captcha, e usecase.Email, signer authUC.TokenSigner, a usecase.UserAuth) {
	r := &V1{auth: a, email: e, captcha: c, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	authPublicGroup := apiV1Group.Group("/auth")
	{
		authPublicGroup.Post("/register", r.register)
		authPublicGroup.Post("/login", r.login)
		authPublicGroup.Post("/refresh", r.refresh)
		authPublicGroup.Post("/forgot", r.forgot)
	}

	authAuthGroup := apiV1Group.Group("/auth", middleware.NewUserJWTMiddleware(signer, a))
	{
		authAuthGroup.Post("/logout", r.logout)
	}
}

func NewUserRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, user usecase.User) {
	r := &V1{auth: auth, user: user, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	userAuthGroup := apiV1Group.Group("/user", middleware.NewUserJWTMiddleware(signer, auth))
	{
		userAuthGroup.Get("/me", r.getMe)
		userAuthGroup.Put("/profile", r.updateProfile)
		userAuthGroup.Put("/password", r.changePassword)
	}
}

func NewContentRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, content usecase.Content) {
	r := &V1{content: content, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	contentPublicGroup := apiV1Group.Group("/content")
	{
		contentPublicGroup.Get("/categories", r.listCategories)
		contentPublicGroup.Get("/tags", r.listTags)
	}

	contentOptionalGroup := apiV1Group.Group("/content", middleware.NewOptionalUserJWTMiddleware(signer, auth))
	{
		contentOptionalGroup.Get("/posts", r.listPosts)
		contentOptionalGroup.Get("/posts/:slug", r.getPost)
	}

	contentAuthGroup := apiV1Group.Group("/content", middleware.NewUserJWTMiddleware(signer, auth))
	{
		contentAuthGroup.Post("/posts/:id/likes", r.togglePostLike)
		contentAuthGroup.Delete("/posts/:id/likes", r.removePostLike)
	}
}

func NewCommentRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, comment usecase.Comment) {
	r := &V1{comment: comment, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	contentOptionalGroup := apiV1Group.Group("/content", middleware.NewOptionalUserJWTMiddleware(signer, auth))
	{
		contentOptionalGroup.Get("/posts/:id/comments", r.listComments)
	}

	contentAuthGroup := apiV1Group.Group("/content", middleware.NewUserJWTMiddleware(signer, auth))
	{
		contentAuthGroup.Post("/posts/:id/comments", r.submitComment)
	}

	commentsAuthGroup := apiV1Group.Group("/comments", middleware.NewUserJWTMiddleware(signer, auth))
	{
		commentsAuthGroup.Post("/:id/likes", r.toggleCommentLike)
		commentsAuthGroup.Delete("/:id/likes", r.removeCommentLike)
		commentsAuthGroup.Delete("/:id", r.deleteComment)
	}
}

func NewFeedbackRoutes(apiV1Group fiber.Router, l logger.Interface, feedback usecase.Feedback) {
	r := &V1{feedback: feedback, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	feedbackPublicGroup := apiV1Group.Group("/feedbacks")
	{
		feedbackPublicGroup.Post("/", r.submitFeedback)
	}
}

func NewLinkRoutes(apiV1Group fiber.Router, l logger.Interface, link usecase.Link) {
	r := &V1{link: link, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	linkPublicGroup := apiV1Group.Group("/links")
	{
		linkPublicGroup.Get("/", r.listLinks)
	}
}

func NewFileRoutes(apiV1Group fiber.Router, l logger.Interface, file usecase.File) {
	r := &V1{file: file, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	filePublicGroup := apiV1Group.Group("/files")
	{
		filePublicGroup.Get("/*", r.getFileURL)
	}
}

func NewSettingRoutes(apiV1Group fiber.Router, l logger.Interface, setting usecase.Setting) {
	r := &V1{setting: setting, logger: l, validate: validator.New(validator.WithRequiredStructEnabled())}

	settingPublicGroup := apiV1Group.Group("/settings")
	{
		settingPublicGroup.Get("/", r.listSettings)
	}
}

func NewNotificationRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, notification usecase.Notification) {
	r := &V1{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), notification: notification, signer: signer}

	notificationPublicGroup := apiV1Group.Group("/notifications")
	{
		notificationPublicGroup.Get("/stream", r.streamNotifications)
	}

	notificationAuthGroup := apiV1Group.Group("/notifications", middleware.NewUserJWTMiddleware(signer, auth))
	{
		notificationAuthGroup.Get("/", r.listNotifications)
		notificationAuthGroup.Get("/unread", r.getUnreadCount)
		notificationAuthGroup.Put("/:id/read", r.markRead)
		notificationAuthGroup.Put("/read-all", r.markAllRead)
		notificationAuthGroup.Delete("/:id", r.deleteNotification)
	}
}
