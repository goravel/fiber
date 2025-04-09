package fiber

import (
	"context"
	"errors"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It does not cancel long running executions. Underlying executions must handle timeout by using context.Context parameter.
// For details, see https://github.com/valyala/fasthttp/issues/965
func Timeout(timeout time.Duration) contractshttp.Middleware {
	return func(next contractshttp.Handler) contractshttp.Handler {
		return contractshttp.HandlerFunc(func(ctx contractshttp.Context) error {
			timeoutCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
			defer cancel()

			ctx.WithContext(timeoutCtx)

			done := make(chan struct{})
			var resp contractshttp.Response

			go func(resp contractshttp.Response) {
				defer func() {
					if err := recover(); err != nil {
						globalRecoverCallback(ctx, err)
					}

					close(done)
				}()
				resp = next.ServeHTTP(ctx)
			}(resp)

			select {
			case <-done:
			case <-timeoutCtx.Done():
				if errors.Is(ctx.Context().Err(), context.DeadlineExceeded) {
					ctx.Request().Abort(contractshttp.StatusRequestTimeout)
				}
			}

			return resp
		})
	}
}
