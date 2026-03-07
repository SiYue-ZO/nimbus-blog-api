package admin

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
)

type Admin struct {
	logger   logger.Interface
	validate *validator.Validate
	sess     *session.Store
	auth     usecase.AdminAuth
	userAuth usecase.UserAuth
	file     usecase.File
	user     usecase.User
	content  usecase.Content
	comment  usecase.Comment
	feedback usecase.Feedback
	link     usecase.Link
	setting  usecase.Setting
	notify   usecase.Notification
}
