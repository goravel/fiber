package fiber

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It does not cancel long running executions. Underlying executions must handle timeout by using context.Context parameter.
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

		done := make(chan struct{})
		var timedOut atomic.Bool

		go func() {
			defer func() {
				if err := recover(); err != nil {
					if !timedOut.Load() {
						globalRecoverCallback(ctx, err)
					}
				}

				close(done)
			}()
			ctx.Request().Next()
		}()

		select {
		case <-done:
		case <-timeoutCtx.Done():
			timedOut.Store(true)
			if errors.Is(ctx.Context().Err(), context.DeadlineExceeded) {
				ctx.Request().Abort(contractshttp.StatusRequestTimeout)
			}
		}
	}
}
