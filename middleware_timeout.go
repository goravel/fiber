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

	err := m.handler(fiberCtx.Instance())

	// After a timeout, Fiber commits a fresh response. On v3.6 that response is 503 and
	// Goravel's contract is 408, so rewrite the committed response rather than
	// the live one, which the abandoned handler may still be writing.
	if timeoutResponse := fiberCtx.Instance().RequestCtx().LastTimeoutErrorResponse(); timeoutResponse != nil {
		timeoutResponse.SetStatusCode(contractshttp.StatusRequestTimeout)
		timeoutResponse.SetBodyString(contractshttp.StatusText(contractshttp.StatusRequestTimeout))
		return
	}

	if err == nil {
		return
	}

	if !errors.Is(err, fiber.ErrRequestTimeout) {
		if err := renderFiberError(fiberCtx.Instance(), err); err != nil {
			panic(err)
		}
	}
}

// Timeout creates middleware to set a timeout for a request.
// NOTICE: Timed-out requests respond with 408 Request Timeout, pinned here because
// Fiber v3.6 changed its timeout middleware default to 503. It relies on Fiber's
// timeout middleware so a timed-out request does not recycle the underlying request
// context into a later request.
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
