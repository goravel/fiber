package fiber

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It does not cancel long running executions. Underlying executions must handle timeout by using context.Context parameter.
// For details, see https://github.com/valyala/fasthttp/issues/965
func Timeout(timeout time.Duration) contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		timeoutCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
		defer cancel()

		ctx.WithContext(timeoutCtx)

		done := make(chan struct{})

		go func() {
			defer func() {
				if err := recover(); err != nil {
					if globalRecoverCallback != nil {
						globalRecoverCallback(ctx, err)
					} else {
						LogFacade.Error(err)
						ctx.Request().AbortWithStatusJson(http.StatusInternalServerError, fiber.Map{"error": "Internal Server Error"})
					}
				}

				close(done)
			}()
			ctx.Request().Next()
		}()

		select {
		case <-done:
		case <-timeoutCtx.Done():
			if errors.Is(ctx.Context().Err(), context.DeadlineExceeded) {
				ctx.Request().AbortWithStatusJson(http.StatusGatewayTimeout, fiber.Map{"error": "Request Timeout"})
			}
		}
	}
}
