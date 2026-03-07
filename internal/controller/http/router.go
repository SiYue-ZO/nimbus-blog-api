package http

import (
	"net/http"
	"time"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/session"
	redisstore "github.com/gofiber/storage/redis/v3"
	"github.com/scc749/nimbus-blog-api/config"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/middleware"
	v1 "github.com/scc749/nimbus-blog-api/internal/controller/http/v1"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
)

func NewRouter(app *fiber.App, cfg *config.Config, l logger.Interface, auth usecase.Auth, captcha usecase.Captcha, email usecase.Email, signer authUC.TokenSigner, file usecase.File, user usecase.User, content usecase.Content, comment usecase.Comment, feedback usecase.Feedback, link usecase.Link, setting usecase.Setting, notificationUC usecase.Notification) {
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type", "Authorization"},
	}))

	// if cfg.Metrics.Enabled {
	// 	prometheus := fiberprometheus.New("nimbus-blog-api")
	// 	prometheus.RegisterAt(app, "/metrics")
	// 	app.Use(prometheus.Middleware)
	// }

	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swaggo.HandlerDefault)
	}

	app.Get("/healthz", func(ctx fiber.Ctx) error { return ctx.SendStatus(http.StatusOK) })

	apiGroup := app.Group("/api")

	apiAdminGroup := apiGroup.Group("/admin")
	{
		rs := redisstore.New(redisstore.Config{Host: cfg.Redis.Host, Port: cfg.Redis.Port, Password: cfg.Redis.Password, Database: cfg.Redis.DB})
		store := session.NewStore(session.Config{Storage: rs, CookieHTTPOnly: true, CookieSecure: true, CookieSameSite: "Strict", IdleTimeout: 24 * time.Hour})

		admin.NewAuthRoutes(apiAdminGroup, l, store, auth.Admin())
		admin.NewFileRoutes(apiAdminGroup, l, store, file)
		admin.NewUserRoutes(apiAdminGroup, l, store, auth.User(), user)
		admin.NewContentRoutes(apiAdminGroup, l, store, content)
		admin.NewCommentRoutes(apiAdminGroup, l, store, comment)
		admin.NewFeedbackRoutes(apiAdminGroup, l, store, feedback)
		admin.NewLinkRoutes(apiAdminGroup, l, store, link)
		admin.NewSettingRoutes(apiAdminGroup, l, store, setting, file)
		admin.NewNotificationRoutes(apiAdminGroup, l, store, notificationUC)
	}

	apiV1Group := apiGroup.Group("/v1")
	{
		v1.NewCaptchaRoutes(apiV1Group, l, captcha)
		v1.NewEmailRoutes(apiV1Group, l, captcha, email)
		v1.NewAuthRoutes(apiV1Group, l, captcha, email, signer, auth.User())
		v1.NewFileRoutes(apiV1Group, l, file)
		v1.NewUserRoutes(apiV1Group, l, signer, auth.User(), user)
		v1.NewContentRoutes(apiV1Group, l, signer, auth.User(), content)
		v1.NewCommentRoutes(apiV1Group, l, signer, auth.User(), comment)
		v1.NewFeedbackRoutes(apiV1Group, l, feedback)
		v1.NewLinkRoutes(apiV1Group, l, link)
		v1.NewSettingRoutes(apiV1Group, l, setting)
		v1.NewNotificationRoutes(apiV1Group, l, signer, auth.User(), notificationUC)
	}
}
