package fiber

import (
	"context"
	"errors"
	"net/http"
	"time"

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
				if r := recover(); r != nil {
					LogFacade.Request(ctx.Request()).Error(r)
					// TODO can be customized in https://github.com/goravel/goravel/issues/521
					ctx.Response().Status(http.StatusInternalServerError).String("Internal Server Error").Render()
				}
				close(done)
			}()

			ctx.Request().Next()
		}()

		select {
		case <-done:
		case <-ctx.Context().Done():
			if errors.Is(ctx.Context().Err(), context.DeadlineExceeded) {
				ctx.Request().AbortWithStatus(http.StatusGatewayTimeout)
			}
		}
	}
}
