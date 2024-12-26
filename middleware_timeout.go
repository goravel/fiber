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
			defer HandleRecover(ctx, globalRecoverCallback)
			ctx.Request().Next()
			close(done)
		}()

		select {
		case <-done:
		case <-timeoutCtx.Done():
			if errors.Is(ctx.Context().Err(), context.DeadlineExceeded) {
				ctx.Request().AbortWithStatus(http.StatusGatewayTimeout)
			}
		}
	}
}
