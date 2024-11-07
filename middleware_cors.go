package fiber

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/goravel/framework/contracts/http"
)

func Cors() http.Middleware {
	return func(ctx http.Context) {
		path := ctx.Request().Path()
		corsPaths, ok := ConfigFacade.Get("cors.paths").([]string)
		if !ok {
			ctx.Request().Next()
			return
		}

		needCors := false
		for _, corsPath := range corsPaths {
			corsPath = pathToFiberPath(corsPath)
			if strings.HasSuffix(corsPath, "*") {
				corsPath = strings.ReplaceAll(corsPath, "*", "")
				if corsPath == "" || strings.HasPrefix(strings.TrimPrefix(path, "/"), strings.TrimPrefix(corsPath, "/")) {
					needCors = true
					break
				}
			} else {
				if strings.TrimPrefix(path, "/") == strings.TrimPrefix(corsPath, "/") {
					needCors = true
					break
				}
			}
		}

		if !needCors {
			ctx.Request().Next()
			return
		}

		fiberCtx := ctx.(*Context)
		if err := cors.New(cors.Config{
			AllowMethods:     allowedMethods(),
			AllowOrigins:     allowedOrigins(),
			AllowHeaders:     allowedHeaders(),
			ExposeHeaders:    exposedHeaders(),
			MaxAge:           ConfigFacade.GetInt("cors.max_age"),
			AllowCredentials: ConfigFacade.GetBool("cors.supports_credentials"),
		})(fiberCtx.Instance()); err != nil {
			panic(err)
		}
	}
}

func allowedMethods() string {
	var allowedMethods string
	allowedMethodConfigs := ConfigFacade.Get("cors.allowed_methods").([]string)
	for i, method := range allowedMethodConfigs {
		if method == "*" {
			allowedMethods = fmt.Sprintf("%s,%s,%s,%s,%s,%s", http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch)
			break
		}
		if i == len(allowedMethodConfigs)-1 {
			allowedMethods += method
			break
		}

		allowedMethods += method + ","
	}

	return allowedMethods
}

func allowedOrigins() string {
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

	return allowedOrigins
}

func allowedHeaders() string {
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

	return allowedHeaders
}

func exposedHeaders() string {
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

	return exposedHeaders
}
