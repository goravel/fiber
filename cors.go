package fiber

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	httpcontract "github.com/goravel/framework/contracts/http"
)

func Cors() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		switch ctx := ctx.(type) {
		case *Context:
			var allowedMethods string
			allowedMethodConfigs := ConfigFacade.Get("cors.allowed_methods").([]string)
			for i, method := range allowedMethodConfigs {
				if method == "*" {
					allowedMethods = "GET,POST,HEAD,PUT,DELETE,PATCH"
					break
				}
				if i == len(allowedMethodConfigs)-1 {
					allowedMethods += method
					break
				}

				allowedMethods += method + ","
			}
			var allowedOrigins string
			allowedOriginConfigs := ConfigFacade.Get("cors.allowed_origins").([]string)
			for i, origin := range allowedOriginConfigs {
				if origin == "*" {
					allowedOrigins = "*"
					break
				}
				if i == len(allowedOriginConfigs)-1 {
					allowedOrigins += origin
					break
				}

				allowedOrigins += origin + ","
			}
			var allowedHeaders string
			allowedHeaderConfigs := ConfigFacade.Get("cors.allowed_headers").([]string)
			for i, header := range allowedHeaderConfigs {
				if header == "*" {
					allowedHeaders = ""
					break
				}
				if i == len(allowedHeaderConfigs)-1 {
					allowedHeaders += header
					break
				}

				allowedHeaders += header + ","
			}
			var exposedHeaders string
			exposedHeaderConfigs := ConfigFacade.Get("cors.exposed_headers").([]string)
			for i, header := range exposedHeaderConfigs {
				if header == "*" {
					exposedHeaders = ""
					break
				}
				if i == len(exposedHeaderConfigs)-1 {
					exposedHeaders += header
					break
				}

				exposedHeaders += header + ","
			}

			_ = cors.New(cors.Config{
				AllowMethods:     allowedMethods,
				AllowOrigins:     allowedOrigins,
				AllowHeaders:     allowedHeaders,
				ExposeHeaders:    exposedHeaders,
				MaxAge:           ConfigFacade.GetInt("cors.max_age"),
				AllowCredentials: ConfigFacade.GetBool("cors.supports_credentials"),
			})(ctx.Instance())

			return
		}

		ctx.Request().Next()
	}
}
