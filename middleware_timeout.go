package fiber

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	fibertimeout "github.com/gofiber/fiber/v3/middleware/timeout"
	contractshttp "github.com/goravel/framework/contracts/http"
)

type timeoutMiddleware struct {
	handler fiber.Handler
}

func (m *timeoutMiddleware) Signature() string {
	return "goravel:timeout"
}

func (m *timeoutMiddleware) Handle(ctx contractshttp.Context) {
	fiberCtx := ctx.(*Context)
	fiberCtx.Instance().SetContext(fiberCtx.Context())

	if err := m.handler(fiberCtx.Instance()); err != nil && !errors.Is(err, fiber.ErrRequestTimeout) {
		if err := renderFiberError(fiberCtx.Instance(), err); err != nil {
			panic(err)
		}
	}
}

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It relies on Fiber's timeout middleware so timed-out requests get a 408 response
// without recycling the underlying request context into a later request.
// For details, see https://github.com/valyala/fasthttp/issues/965
func Timeout(timeout time.Duration) contractshttp.Middleware {
	if timeout <= 0 {
		return &timeoutMiddleware{handler: func(c fiber.Ctx) error {
			return c.Next()
		}}
	}

	handler := fibertimeout.New(func(c fiber.Ctx) (err error) {
		// Mirror Fiber's timeout-aware context into Goravel's request context so
		// downstream handlers observe the same deadline and cancellation signal.
		c.Locals(sharedUserCtxKey, c.Context())

		defer func() {
			if recovered := recover(); recovered != nil {
				if !errors.Is(c.Context().Err(), context.DeadlineExceeded) {
					recoverCtx := NewContext(c)
					defer releaseContext(recoverCtx)
					globalRecoverCallback(recoverCtx, recovered)
				}
				err = nil
			}
		}()

		return c.Next()
	}, fibertimeout.Config{Timeout: timeout})

	return &timeoutMiddleware{handler: handler}
}
