package fiber

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It relies on Fiber's timeout-aware context and fasthttp's timeout response
// path so timed-out requests don't leak their stale response into later requests.
// For details, see https://github.com/valyala/fasthttp/issues/965
func Timeout(timeout time.Duration) contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		if timeout <= 0 {
			ctx.Request().Next()
			return
		}

		timeoutCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
		defer cancel()

		ctx.WithContext(timeoutCtx)

		fiberCtx := ctx.(*Context).Instance()
		detachedCtx := cloneFiberContext(ctx.(*Context))
		detachedInstance := detachedCtx.Instance()
		done := make(chan struct{})

		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					if timeoutCtx.Err() == nil {
						globalRecoverCallback(detachedCtx, recovered)
					}
				}

				detachedInstance.App().ReleaseCtx(detachedInstance)
				releaseContext(detachedCtx)
				close(done)
			}()

			detachedCtx.Request().Next()
		}()

		select {
		case <-done:
		case <-timeoutCtx.Done():
			if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				if err := fiberCtx.Status(fiber.ErrRequestTimeout.Code).SendString(fiber.ErrRequestTimeout.Message); err != nil {
					panic(err)
				}
				fiberCtx.Context().TimeoutErrorWithResponse(fiberCtx.Response())
			}
		}
	}
}
