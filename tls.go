package fiber

import (
	"strings"

	httpcontract "github.com/goravel/framework/contracts/http"
)

func Tls() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		host := ConfigFacade.GetString("http.tls.host")
		port := ConfigFacade.GetString("http.tls.port")
		cert := ConfigFacade.GetString("http.tls.ssl.cert")
		key := ConfigFacade.GetString("http.tls.ssl.key")

		if host == "" || cert == "" || key == "" || ctx.Request().Origin().TLS == nil {
			ctx.Request().Next()

			return
		}

		completeHost := host
		if port != "" {
			completeHost = host + ":" + port
		}

		if strings.HasPrefix(ctx.Request().FullUrl(), "http://") {
			url := "https://" + completeHost + ctx.Request().Url()
			ctx.Response().Redirect(httpcontract.StatusFound, url)
		}

		ctx.Request().Next()
	}
}
