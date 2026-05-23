package fiber

import (
	"context"
	"errors"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/gofiber/fiber/v3"
	fibertimeout "github.com/gofiber/fiber/v3/middleware/timeout"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It relies on Fiber's timeout middleware so timed-out requests get a 408 response
// without recycling the underlying request context into a later request.
// For details, see https://github.com/valyala/fasthttp/issues/965
func Timeout(timeout time.Duration) contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		if timeout <= 0 {
			ctx.Request().Next()
			return
		}

		fiberCtx, ok := ctx.(*Context)
		if !ok {
			timeoutCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
			defer cancel()

			ctx.WithContext(timeoutCtx)
			ctx.Request().Next()
			return
		}

		handler := fibertimeout.New(func(c fiber.Ctx) (err error) {
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

		if err := handler(fiberCtx.Instance()); err != nil && !errors.Is(err, fiber.ErrRequestTimeout) {
			if err := renderFiberError(fiberCtx.Instance(), err); err != nil {
				panic(err)
			}
		}
	}
}
