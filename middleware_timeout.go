package fiber

import (
	"context"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
)

// Timeout creates middleware to set a timeout for a request.
// NOTICE: It does not interrupt long running executions. Underlying executions must observe ctx.Done() / ctx.Err() and stop cooperatively.
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
		ctx.Request().Next()
	}
}
