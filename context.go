package fiber

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

type contextKeyType struct{}

type sessionKeyType struct{}

type userContextKeyType struct{}

var (
	contextKey          = contextKeyType{}
	sessionKey          = sessionKeyType{}
	userContextKey      = userContextKeyType{}
	internalContextKeys = []any{
		contextKey,
		sessionKey,
		userContextKey,
	}
)

func Background() http.Context {
	app := fiber.New()
	httpCtx := app.AcquireCtx(&fasthttp.RequestCtx{})

	return NewContext(httpCtx)
}

var contextPool = sync.Pool{New: func() any {
	return &Context{}
}}

type Context struct {
	instance *fiber.Ctx
	request  http.ContextRequest
	response http.ContextResponse
}

func NewContext(c *fiber.Ctx) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.instance = c
	return ctx
}

func (c *Context) Request() http.ContextRequest {
	if c.request == nil {
		request := NewContextRequest(c, LogFacade, ValidationFacade)
		c.request = request
	}

	return c.request
}

func (c *Context) Response() http.ContextResponse {
	if c.response == nil {
		response := NewContextResponse(c.instance, &ResponseOrigin{Ctx: c.instance})
		c.response = response
	}

	return c.response
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
