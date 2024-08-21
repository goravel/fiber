package fiber

import (
	"strings"

	"github.com/gofiber/fiber/v3/middleware/cors"
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

func allowedMethods() []string {
	var allowedMethods []string
	allowedMethodConfigs := ConfigFacade.Get("cors.allowed_methods").([]string)
	for _, method := range allowedMethodConfigs {
		if method == "*" {
			allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch}
			break
		}
		allowedMethods = append(allowedMethods, method)
	}

	return allowedMethods
}

func allowedOrigins() []string {
	return ConfigFacade.Get("cors.allowed_origins").([]string)
}

func allowedHeaders() []string {
	var allowedHeaders []string
	allowedHeaderConfigs := ConfigFacade.Get("cors.allowed_headers").([]string)
	for _, header := range allowedHeaderConfigs {
		if header == "*" {
			break
		}
		allowedHeaders = append(allowedHeaders, header)
	}

	return allowedHeaders
}

func exposedHeaders() []string {
	var exposedHeaders []string
	exposedHeaderConfigs := ConfigFacade.Get("cors.exposed_headers").([]string)
	for _, header := range exposedHeaderConfigs {
		if header == "*" {
			break
		}
		exposedHeaders = append(exposedHeaders, header)
	}

	return exposedHeaders
}
