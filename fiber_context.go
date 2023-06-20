package fiber

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/goravel/framework/contracts/http"
)

type FiberContext struct {
	instance *fiber.Ctx
	request  http.Request
}

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
	responseOrigin := c.Value("responseOrigin")
	if responseOrigin != nil {
		return NewFiberResponse(c.instance, responseOrigin.(http.ResponseOrigin))
	}

	return NewFiberResponse(c.instance, &BodyWriter{Writer: c.instance.Response().BodyWriter()})
}

func (c *FiberContext) WithValue(key string, value any) {
	ctx := c.instance.UserContext()
	ctx = context.WithValue(ctx, key, value)
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
	return c.instance.UserContext().Value(key)
}

func (c *FiberContext) Instance() *fiber.Ctx {
	return c.instance
}
