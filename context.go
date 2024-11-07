package fiber

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

func Background() http.Context {
	app := fiber.New()
	httpCtx := app.AcquireCtx(&fasthttp.RequestCtx{})

	return NewContext(httpCtx)
}

type Context struct {
	instance *fiber.Ctx
	request  http.ContextRequest
}

func NewContext(ctx *fiber.Ctx) http.Context {
	return &Context{instance: ctx}
}

func (c *Context) Request() http.ContextRequest {
	if c.request == nil {
		c.request = NewContextRequest(c, LogFacade, ValidationFacade)
	}

	return c.request
}

func (c *Context) Response() http.ContextResponse {
	return NewContextResponse(c.instance, &ResponseOrigin{Ctx: c.instance})
}

func (c *Context) WithValue(key any, value any) {
	ctx := context.WithValue(c.instance.UserContext(), key, value)
	c.instance.SetUserContext(ctx)
}

func (c *Context) WithContext(ctx context.Context) {
	c.instance.SetUserContext(ctx)
}

func (c *Context) Context() context.Context {
	return c.instance.UserContext()
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.instance.UserContext().Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.instance.UserContext().Done()
}

func (c *Context) Err() error {
	return c.instance.UserContext().Err()
}

func (c *Context) Value(key any) any {
	return c.instance.UserContext().Value(key)
}

func (c *Context) Instance() *fiber.Ctx {
	return c.instance
}
