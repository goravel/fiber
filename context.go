package fiber

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

const (
	userContextKey = "goravel_userContextKey"
	contextKey     = "goravel_contextKey"
	sessionKey     = "goravel_session"
)

var (
	internalContextKeys = []any{
		contextKey,
		sessionKey,
	}
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
	// Not store the value in the context directly, because we want to return the value map when calling `Context()`.
	values := c.getGoravelContextValues()
	if values == nil {
		values = make(map[any]any)
	}
	values[key] = value
	ctx := context.WithValue(c.instance.UserContext(), contextKey, values)
	c.instance.SetUserContext(ctx)
}

func (c *Context) WithContext(ctx context.Context) {
	// We want to return the original context back when calling `Context()`, so we need to store it.
	ctx = context.WithValue(ctx, userContextKey, ctx)
	ctx = context.WithValue(ctx, contextKey, c.getGoravelContextValues())
	c.instance.SetUserContext(ctx)
}

func (c *Context) Context() context.Context {
	ctx := c.getUserContext()
	values := c.getGoravelContextValues()
	for key, value := range values {
		skip := false
		for _, internalContextKey := range internalContextKeys {
			if key == internalContextKey {
				skip = true
			}
		}

		if !skip {
			ctx = context.WithValue(ctx, key, value)
		}
	}

	return ctx
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
	values := c.getGoravelContextValues()
	if value, exist := values[key]; exist {
		return value
	}

	return c.instance.UserContext().Value(key)
}

func (c *Context) Instance() *fiber.Ctx {
	return c.instance
}

func (c *Context) getGoravelContextValues() map[any]any {
	value := c.instance.UserContext().Value(contextKey)
	if goravelCtxVal, ok := value.(map[any]any); ok {
		return goravelCtxVal
	}

	return make(map[any]any)
}

func (c *Context) getUserContext() context.Context {
	ctx, exist := c.instance.UserContext().Value(userContextKey).(context.Context)
	if !exist {
		ctx = context.Background()
	}

	return ctx
}
