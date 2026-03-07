//go:build wireinject
// +build wireinject

// Package app 应用装配与生命周期管理。

package app

import (
	"context"
	"time"

	"github.com/google/wire"
	minioSDK "github.com/minio/minio-go/v7"
	"github.com/scc749/nimbus-blog-api/config"
	httpctrl "github.com/scc749/nimbus-blog-api/internal/controller/http"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/cache"
	"github.com/scc749/nimbus-blog-api/internal/repo/messaging"
	reponotif "github.com/scc749/nimbus-blog-api/internal/repo/notification"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence"
	"github.com/scc749/nimbus-blog-api/internal/repo/storage"
	"github.com/scc749/nimbus-blog-api/internal/repo/viewbuffer"
	"github.com/scc749/nimbus-blog-api/internal/repo/webapi"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/auth"
	authadmin "github.com/scc749/nimbus-blog-api/internal/usecase/auth/admin"
	authuser "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/internal/usecase/captcha"
	"github.com/scc749/nimbus-blog-api/internal/usecase/comment"
	"github.com/scc749/nimbus-blog-api/internal/usecase/content"
	"github.com/scc749/nimbus-blog-api/internal/usecase/email"
	"github.com/scc749/nimbus-blog-api/internal/usecase/feedback"
	"github.com/scc749/nimbus-blog-api/internal/usecase/file"
	"github.com/scc749/nimbus-blog-api/internal/usecase/link"
	notification "github.com/scc749/nimbus-blog-api/internal/usecase/notification"
	"github.com/scc749/nimbus-blog-api/internal/usecase/setting"
	"github.com/scc749/nimbus-blog-api/internal/usecase/user"
	"github.com/scc749/nimbus-blog-api/pkg/httpserver"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
	minioPkg "github.com/scc749/nimbus-blog-api/pkg/minio"
	"github.com/scc749/nimbus-blog-api/pkg/postgres"
	"github.com/scc749/nimbus-blog-api/pkg/redis"
	"github.com/scc749/nimbus-blog-api/pkg/ssehub"
)

// App 应用容器。

type App struct {
	Info AppInfo

	Logger     logger.Interface
	Postgres   *postgres.Postgres
	Redis      *redis.Redis
	HTTPServer *httpserver.Server
}

// AppInfo 应用信息。
type AppInfo struct {
	Name    string
	Version string
}

// NewApp 创建 App。
func NewApp(info AppInfo, l logger.Interface, pg *postgres.Postgres, r *redis.Redis, srv *httpserver.Server) *App {
	return &App{
		Info:       info,
		Logger:     l,
		Postgres:   pg,
		Redis:      r,
		HTTPServer: srv,
	}
}

// NewAppInfo 创建 AppInfo。
func NewAppInfo(cfg *config.Config) AppInfo {
	return AppInfo{Name: cfg.App.Name, Version: cfg.App.Version}
}

// Infrastructure 基础设施构造。

// NewLogger 创建 Logger。
func NewLogger(cfg *config.Config) logger.Interface {
	return logger.New(cfg.Log.Level)
}

// NewPostgres 创建 Postgres 连接并返回 cleanup。
func NewPostgres(cfg *config.Config) (*postgres.Postgres, func(), error) {
	pg, err := postgres.New(
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
		cfg.Postgres.TimeZone,
		postgres.WithMaxIdleConns(cfg.Postgres.MaxIdleConns),
		postgres.WithMaxOpenConns(cfg.Postgres.MaxOpenConns),
	)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { pg.Close() }
	return pg, cleanup, nil
}

// NewRedis 创建 Redis 连接并返回 cleanup。
func NewRedis(cfg *config.Config) (*redis.Redis, func(), error) {
	rdb, err := redis.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { rdb.Close() }
	return rdb, cleanup, nil
}

// NewMinioClient 创建 MinIO Client 并确保默认 bucket 存在。
func NewMinioClient(cfg *config.Config) (*minioSDK.Client, error) {
	cli, err := minioPkg.New(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.UseSSL,
		minioPkg.WithDefaultBucket(cfg.MinIO.Bucket),
		minioPkg.WithRegion(cfg.MinIO.Region),
	)
	if err != nil {
		return nil, err
	}
	if cfg.MinIO.Bucket != "" {
		_ = cli.EnsureBucket(context.Background(), cfg.MinIO.Bucket)
	}
	return cli.CLI, nil
}

// Persistence Repo（PostgreSQL）。

func NewAdminRepo(pg *postgres.Postgres) repo.AdminRepo {
	return persistence.NewAdminRepo(pg.DB)
}

func NewUserRepo(pg *postgres.Postgres) repo.UserRepo {
	return persistence.NewUserRepo(pg.DB)
}

func NewPostRepo(pg *postgres.Postgres) repo.PostRepo {
	return persistence.NewPostRepo(pg.DB)
}

func NewTagRepo(pg *postgres.Postgres) repo.TagRepo {
	return persistence.NewTagRepo(pg.DB)
}

func NewCategoryRepo(pg *postgres.Postgres) repo.CategoryRepo {
	return persistence.NewCategoryRepo(pg.DB)
}

func NewCommentRepo(pg *postgres.Postgres) repo.CommentRepo {
	return persistence.NewCommentRepo(pg.DB)
}

func NewPostLikeRepo(pg *postgres.Postgres) repo.PostLikeRepo {
	return persistence.NewPostLikeRepo(pg.DB)
}

func NewCommentLikeRepo(pg *postgres.Postgres) repo.CommentLikeRepo {
	return persistence.NewCommentLikeRepo(pg.DB)
}

func NewFeedbackRepo(pg *postgres.Postgres) repo.FeedbackRepo {
	return persistence.NewFeedbackRepo(pg.DB)
}

func NewLinkRepo(pg *postgres.Postgres) repo.LinkRepo {
	return persistence.NewLinkRepo(pg.DB)
}

func NewSiteSettingRepo(pg *postgres.Postgres) repo.SiteSettingRepo {
	return persistence.NewSiteSettingRepo(pg.DB)
}

func NewFileRepo(pg *postgres.Postgres) repo.FileRepo {
	return persistence.NewFileRepo(pg.DB)
}

func NewNotificationRepo(pg *postgres.Postgres) repo.NotificationRepo {
	return persistence.NewNotificationRepo(pg.DB)
}

func NewRefreshTokenBlacklistRepo(pg *postgres.Postgres) repo.RefreshTokenBlacklistRepo {
	return persistence.NewRefreshTokenBlacklistPostgres(pg.DB)
}

// ViewBuffer Repo（浏览量缓冲）。

func NewPostViewRepo(pg *postgres.Postgres, l logger.Interface) (repo.PostViewRepo, func()) {
	return viewbuffer.New(pg.DB, l)
}

// Cache Repo（Redis）。

func NewCaptchaStore(r *redis.Redis) repo.CaptchaStore {
	return cache.NewCaptchaRedisStore(r, 5*time.Minute)
}

func NewEmailCodeStore(r *redis.Redis) repo.EmailCodeStore {
	return cache.NewEmailCodeRedisStore(r, 10*time.Minute)
}

func NewRefreshTokenStore(r *redis.Redis) repo.RefreshTokenStore {
	return cache.NewRefreshTokenRedisStore(r)
}

func NewAdminTwoFASetupStore(r *redis.Redis) repo.AdminTwoFASetupStore {
	return cache.NewAdminTwoFARedisStore(r)
}

// Storage Repo（MinIO）。

func NewObjectStore(cli *minioSDK.Client) repo.ObjectStore {
	return storage.NewMinioStore(cli)
}

// Messaging Repo（SMTP）。

func NewEmailSender(cfg *config.Config) repo.EmailSender {
	return messaging.NewSMTPEmailSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
	)
}

// WebAPI Repo（外部 API）。

func NewTranslationWebAPI() repo.TranslationWebAPI {
	return webapi.NewTranslationWebAPI()
}

func NewLLMWebAPI(cfg *config.Config) repo.LLMWebAPI {
	return webapi.NewLLMWebAPI(cfg.OpenAI.APIKey, cfg.OpenAI.BaseURL, cfg.OpenAI.Model)
}

// Auth UseCase。

func NewTokenSigner(cfg *config.Config) (authuser.TokenSigner, error) {
	return authuser.NewTokenSigner(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
		cfg.JWT.Issuer,
	)
}

func NewAdminAuthUseCase(cfg *config.Config, adminRepo repo.AdminRepo, twoFASetupStore repo.AdminTwoFASetupStore) usecase.AdminAuth {
	totpCfg := authadmin.TOTPConfig{QRWidth: cfg.TwoFA.QRWidth, QRHeight: cfg.TwoFA.QRHeight}
	enc := authadmin.NewEncryptorFromSecret(cfg.TwoFA.EncryptionKey)
	return authadmin.New(adminRepo, twoFASetupStore, cfg.App.Name, authadmin.NewTOTPProviderWithConfig(totpCfg), enc)
}

func NewUserAuthUseCase(userRepo repo.UserRepo, signer authuser.TokenSigner, refreshStore repo.RefreshTokenStore, refreshBlacklist repo.RefreshTokenBlacklistRepo) usecase.UserAuth {
	return authuser.New(userRepo, signer, refreshStore, refreshBlacklist)
}

func NewAuthUseCase(adminAuth usecase.AdminAuth, userAuth usecase.UserAuth) usecase.Auth {
	return auth.New(adminAuth, userAuth)
}

// Captcha UseCase。

func NewCaptchaGenerator(cfg *config.Config) captcha.Generator {
	return captcha.NewBase64Generator(captcha.Config{
		Height:   cfg.Captcha.Height,
		Width:    cfg.Captcha.Width,
		Length:   cfg.Captcha.Length,
		MaxSkew:  cfg.Captcha.MaxSkew,
		DotCount: cfg.Captcha.DotCount,
	})
}

func NewCaptchaUseCase(gen captcha.Generator, store repo.CaptchaStore) usecase.Captcha {
	return captcha.New(store, gen)
}

// Email UseCase。

func NewEmailUseCase(sender repo.EmailSender, codeStore repo.EmailCodeStore) usecase.Email {
	return email.New(sender, codeStore)
}

// File UseCase。

func NewFileUseCase(cfg *config.Config, objectStore repo.ObjectStore, fileRepo repo.FileRepo) usecase.File {
	return file.New(objectStore, fileRepo, cfg.MinIO.Bucket)
}

// User UseCase。

func NewUserUseCase(userRepo repo.UserRepo) usecase.User {
	return user.New(userRepo)
}

// Content UseCase。

func NewContentUseCase(translationAPI repo.TranslationWebAPI, llmAPI repo.LLMWebAPI, adminRepo repo.AdminRepo, postRepo repo.PostRepo, tagRepo repo.TagRepo, categoryRepo repo.CategoryRepo, postLikeRepo repo.PostLikeRepo, fileRepo repo.FileRepo, postViewRepo repo.PostViewRepo) usecase.Content {
	return content.New(translationAPI, llmAPI, adminRepo, postRepo, tagRepo, categoryRepo, postLikeRepo, fileRepo, postViewRepo, content.NewCalculator())
}

// Comment UseCase。

func NewCommentUseCase(commentRepo repo.CommentRepo, commentLikeRepo repo.CommentLikeRepo, userRepo repo.UserRepo, postRepo repo.PostRepo, notifier repo.Notifier) usecase.Comment {
	return comment.New(commentRepo, commentLikeRepo, userRepo, postRepo, notifier)
}

// Feedback UseCase。

func NewFeedbackUseCase(feedbackRepo repo.FeedbackRepo) usecase.Feedback {
	return feedback.New(feedbackRepo)
}

// Link UseCase。

func NewLinkUseCase(linkRepo repo.LinkRepo) usecase.Link {
	return link.New(linkRepo)
}

// Setting UseCase。

func NewSettingUseCase(settingRepo repo.SiteSettingRepo) usecase.Setting {
	return setting.New(settingRepo)
}

// SSEHub SSE Hub。

func NewSSEHub() *ssehub.Hub {
	return ssehub.New()
}

// Notifier 通知推送实现。

func NewNotifier(notificationRepo repo.NotificationRepo, hub *ssehub.Hub) repo.Notifier {
	return reponotif.NewNotifier(notificationRepo, hub)
}

// Notification UseCase。

func NewNotificationUseCase(notificationRepo repo.NotificationRepo, notifier repo.Notifier, hub *ssehub.Hub) usecase.Notification {
	return notification.New(notificationRepo, notifier, hub)
}

// HTTP HTTP Server。

func SetupHTTPServer(
	cfg *config.Config, l logger.Interface,
	auth usecase.Auth, captchaUC usecase.Captcha, emailUC usecase.Email, signer authuser.TokenSigner,
	fileUC usecase.File, userUC usecase.User, contentUC usecase.Content, commentUC usecase.Comment,
	feedbackUC usecase.Feedback, linkUC usecase.Link, settingUC usecase.Setting,
	notificationUC usecase.Notification,
) *httpserver.Server {
	srv := httpserver.New(l, httpserver.WithPort(cfg.HTTP.Port), httpserver.WithPrefork(cfg.HTTP.UsePreforkMode))
	httpctrl.NewRouter(srv.App, cfg, l, auth, captchaUC, emailUC, signer, fileUC, userUC, contentUC, commentUC, feedbackUC, linkUC, settingUC, notificationUC)
	return srv
}

// ProviderSet Wire ProviderSet。
var ProviderSet = wire.NewSet(
	// App 应用容器。
	NewAppInfo,
	NewLogger,
	NewApp,
	// Infrastructure 基础设施。
	NewPostgres,
	NewRedis,
	NewMinioClient,
	// RepoPersistence Postgres Repo。
	NewAdminRepo,
	NewUserRepo,
	NewPostRepo,
	NewTagRepo,
	NewCategoryRepo,
	NewCommentRepo,
	NewPostLikeRepo,
	NewCommentLikeRepo,
	NewFeedbackRepo,
	NewLinkRepo,
	NewSiteSettingRepo,
	NewFileRepo,
	NewNotificationRepo,
	NewRefreshTokenBlacklistRepo,
	// RepoViewBuffer 浏览量缓冲。
	NewPostViewRepo,
	// RepoCache Redis Repo。
	NewCaptchaStore,
	NewEmailCodeStore,
	NewRefreshTokenStore,
	NewAdminTwoFASetupStore,
	// RepoStorage MinIO Repo。
	NewObjectStore,
	// RepoMessaging SMTP Repo。
	NewEmailSender,
	// RepoWebAPI 外部 API。
	NewTranslationWebAPI,
	NewLLMWebAPI,
	// UseCaseAuth 认证用例。
	NewTokenSigner,
	NewAdminAuthUseCase,
	NewUserAuthUseCase,
	NewAuthUseCase,
	// UseCaseCaptcha 验证码用例。
	NewCaptchaGenerator,
	NewCaptchaUseCase,
	// UseCaseEmail 邮件用例。
	NewEmailUseCase,
	// UseCaseFile 文件用例。
	NewFileUseCase,
	// UseCaseUser 用户用例。
	NewUserUseCase,
	// UseCaseContent 内容用例。
	NewContentUseCase,
	// UseCaseComment 评论用例。
	NewCommentUseCase,
	// UseCaseFeedback 反馈用例。
	NewFeedbackUseCase,
	// UseCaseLink 友链用例。
	NewLinkUseCase,
	// UseCaseSetting 设置用例。
	NewSettingUseCase,
	// Pkg 基础包。
	NewSSEHub,
	// RepoNotifier 通知推送。
	NewNotifier,
	// UseCaseNotification 通知用例。
	NewNotificationUseCase,
	// HTTP HTTP Server。
	SetupHTTPServer,
)

// InitializeApp 初始化 App 并返回 cleanup。
func InitializeApp(cfg *config.Config) (*App, func(), error) {
	wire.Build(ProviderSet)
	return nil, nil, nil
}
