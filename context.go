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
type sharedUserCtxKeyType struct{}

var (
	sessionKey       = sessionKeyType{}
	sharedValuesKey  = sharedValuesKeyType{}
	sharedUserCtxKey = sharedUserCtxKeyType{}
)

func Background() http.Context {
	app := fiber.New()
	httpCtx := app.AcquireCtx(&fasthttp.RequestCtx{})

	return NewContext(httpCtx)
}

var contextPool = sync.Pool{New: func() any {
	return &Context{}
}}

// sharedValues holds the request values that every *Context wrapper of one
// fiber.Ctx shares. The mutex lives here, not in Context, because middleware
// and handler each get their own *Context over the same map.
type sharedValues struct {
	mu     sync.RWMutex
	values map[any]any
}

func newSharedValues() *sharedValues {
	return &sharedValues{values: make(map[any]any)}
}

func (s *sharedValues) load(key any) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.values[key]
	return value, ok
}

func (s *sharedValues) store(key any, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.values[key] = value
}

// clone returns a consistent copy so callers can iterate without holding the
// lock, and returns nil when there is nothing to copy.
func (s *sharedValues) clone() map[any]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.values) == 0 {
		return nil
	}
	values := make(map[any]any, len(s.values))
	for key, value := range s.values {
		values[key] = value
	}
	return values
}

type Context struct {
	instance fiber.Ctx
	request  http.ContextRequest
	response http.ContextResponse
	userCtx  context.Context
	values   *sharedValues
}

func NewContext(c fiber.Ctx) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.instance = c
	if existing, ok := c.Locals(sharedUserCtxKey).(context.Context); ok {
		ctx.userCtx = existing
	} else {
		ctx.userCtx = nil
	}
	if existing, ok := c.Locals(sharedValuesKey).(*sharedValues); ok {
		ctx.values = existing
	} else {
		ctx.values = newSharedValues()
		c.Locals(sharedValuesKey, ctx.values)
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
		c.values = newSharedValues()
		c.instance.Locals(sharedValuesKey, c.values)
	}
	c.values.store(key, value)
}

func (c *Context) WithContext(ctx context.Context) {
	c.userCtx = ctx
	c.instance.Locals(sharedUserCtxKey, ctx)
}

func (c *Context) Context() context.Context {
	ctx := c.userCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if c.values == nil {
		return ctx
	}
	for key, value := range c.values.clone() {
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
		if value, ok := c.values.load(key); ok {
			return value
		}
	}
	return c.Context().Value(key)
}

func (c *Context) Instance() fiber.Ctx {
	return c.instance
}
