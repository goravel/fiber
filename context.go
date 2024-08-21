package fiber

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

func Background() http.Context {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})

	return &Context{instance: ctx}
}

var contextPool = sync.Pool{New: func() any {
	return &Context{}
}}

type Context struct {
	instance fiber.Ctx
	request  http.ContextRequest
	response http.ContextResponse
}

func (c *Context) Request() http.ContextRequest {
	if c.request == nil {
		request := contextRequestPool.Get().(*ContextRequest)
		httpBody, err := getHttpBody(c)
		if err != nil {
			LogFacade.Error(fmt.Sprintf("%+v", errors.Unwrap(err)))
		}
		request.ctx = c
		request.instance = c.instance
		request.httpBody = httpBody
		c.request = request
	}

	return c.request
}

func (c *Context) Response() http.ContextResponse {
	if c.response == nil {
		response := contextResponsePool.Get().(*ContextResponse)
		response.instance = c.instance
		response.origin = &ResponseOrigin{Ctx: c.instance}
		c.response = response
	}

	return c.response
}

func (c *Context) WithValue(key string, value any) {
	ctx := context.WithValue(c.instance.UserContext(), key, value)
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
	if keyStr, ok := key.(string); ok {
		return c.instance.UserContext().Value(keyStr)
	}

	return nil
}

func (c *Context) Instance() fiber.Ctx {
	return c.instance
}
