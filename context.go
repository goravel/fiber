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
type requestValuesKeyType struct{}
type sharedUserCtxKeyType struct{}

var (
	sessionKey       = sessionKeyType{}
	requestValuesKey = requestValuesKeyType{}
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

// requestValues holds the request values that every *Context wrapper of one
// fiber.Ctx shares. The mutex lives here, not in Context, because middleware
// and handler each get their own *Context over the same map.
type requestValues struct {
	mu     sync.RWMutex
	values map[any]any
}

type requestValue struct {
	key   any
	value any
}

func newRequestValues() *requestValues {
	return &requestValues{}
}

func (s *requestValues) load(key any) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.values[key]
	return value, ok
}

func (s *requestValues) store(key any, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.values == nil {
		s.values = make(map[any]any)
	}
	s.values[key] = value
}

// snapshot copies the entries under the read lock so the context chain can be
// built without holding it.
func (s *requestValues) snapshot() []requestValue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]requestValue, 0, len(s.values))
	for key, value := range s.values {
		entries = append(entries, requestValue{key: key, value: value})
	}
	return entries
}

type Context struct {
	instance fiber.Ctx
	request  http.ContextRequest
	response http.ContextResponse
	userCtx  context.Context
	values   *requestValues
}

func NewContext(c fiber.Ctx) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.instance = c
	if existing, ok := c.Locals(sharedUserCtxKey).(context.Context); ok {
		ctx.userCtx = existing
	} else {
		ctx.userCtx = nil
	}
	if existing, ok := c.Locals(requestValuesKey).(*requestValues); ok {
		ctx.values = existing
	} else {
		ctx.values = newRequestValues()
		c.Locals(requestValuesKey, ctx.values)
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
		// Only NewContext produces a usable Context: c.instance is nil on a zero value.
		c.values = newRequestValues()
		if c.instance != nil {
			c.instance.Locals(requestValuesKey, c.values)
		}
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
	rv := c.values
	if rv == nil {
		return ctx
	}
	for _, entry := range rv.snapshot() {
		if entry.key == sessionKey {
			continue
		}
		ctx = context.WithValue(ctx, entry.key, entry.value)
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
