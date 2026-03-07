package shared

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	codes "github.com/scc749/nimbus-blog-api/internal/controller/http/bizcode"
)

type Option func(*Envelope)

func WithMsg(msg string) Option        { return func(e *Envelope) { e.Message = msg } }
func WithData(data interface{}) Option { return func(e *Envelope) { e.Data = data } }
func WithCode(code string) Option      { return func(e *Envelope) { e.Code = code } }

func WriteSuccess(ctx fiber.Ctx, opts ...Option) error {
	env := Envelope{Code: codes.Success, Message: "ok"}
	for _, opt := range opts {
		opt(&env)
	}
	return ctx.Status(http.StatusOK).JSON(env)
}

func WriteError(ctx fiber.Ctx, httpCode int, bizCode string, msg string) error {
	return ctx.Status(httpCode).JSON(Envelope{Code: bizCode, Message: msg})
}
