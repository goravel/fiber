package fiber

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

type sessionKeyType struct{}
type sharedValuesKeyType struct{}

var (
	sessionKey      = sessionKeyType{}
	sharedValuesKey = sharedValuesKeyType{}
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
	instance fiber.Ctx
	request  http.ContextRequest
	response http.ContextResponse
	userCtx  context.Context
	values   map[any]any
}

func NewContext(c fiber.Ctx) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.instance = c
	ctx.userCtx = nil
	if existing, ok := c.Locals(sharedValuesKey).(map[any]any); ok {
		ctx.values = existing
	} else {
		ctx.values = nil
	}
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
		response := NewContextResponse(c.instance, &ResponseOrigin{Ctx: c.instance}, c)
		c.response = response
	}

	return c.response
}

func (c *Context) WithValue(key any, value any) {
	if c.values == nil {
		c.values = make(map[any]any)
		c.instance.Locals(sharedValuesKey, c.values)
	}
	c.values[key] = value
}

func (c *Context) WithContext(ctx context.Context) {
	c.userCtx = ctx
}

func (c *Context) Context() context.Context {
	ctx := c.userCtx
	if ctx == nil {
		ctx = context.Background()
	}
	for key, value := range c.values {
		if key == sessionKey {
			continue
		}
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Context().Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *Context) Err() error {
	return c.Context().Err()
}

func (c *Context) Value(key any) any {
	if c.values != nil {
		if v, ok := c.values[key]; ok {
			return v
		}
	}
	return c.Context().Value(key)
}

func (c *Context) Instance() fiber.Ctx {
	return c.instance
}
