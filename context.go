package fiber

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/goravel/framework/contracts/http"
)

func Background() http.Context {
	app := fiber.New()
	httpCtx := app.AcquireCtx(&fasthttp.RequestCtx{})

	return NewFiberContext(httpCtx)
}

type FiberContext struct {
	instance *fiber.Ctx
	request  http.Request
}

type ctxKey string

func NewFiberContext(ctx *fiber.Ctx) http.Context {
	return &FiberContext{instance: ctx}
}

func (c *FiberContext) Request() http.Request {
	if c.request == nil {
		c.request = NewFiberRequest(c, LogFacade, ValidationFacade)
	}

	return c.request
}

func (c *FiberContext) Response() http.Response {
	return NewFiberResponse(c.instance, &ResponseOrigin{Ctx: c.instance})
}

func (c *FiberContext) WithValue(key string, value any) {
	ctx := context.WithValue(c.instance.UserContext(), ctxKey(key), value)
	c.instance.SetUserContext(ctx)
}

func (c *FiberContext) Context() context.Context {
	return c.instance.UserContext()
}

func (c *FiberContext) Deadline() (deadline time.Time, ok bool) {
	return c.instance.UserContext().Deadline()
}

func (c *FiberContext) Done() <-chan struct{} {
	return c.instance.UserContext().Done()
}

func (c *FiberContext) Err() error {
	return c.instance.UserContext().Err()
}

func (c *FiberContext) Value(key any) any {
	if keyStr, ok := key.(string); ok {
		return c.instance.UserContext().Value(ctxKey(keyStr))
	}

	return nil
}

func (c *FiberContext) Instance() *fiber.Ctx {
	return c.instance
}
