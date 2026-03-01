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
		handler := cors.New(cors.Config{
			AllowMethods:     allowedMethods(),
			AllowOrigins:     allowedOrigins(),
			AllowHeaders:     allowedHeaders(),
			ExposeHeaders:    exposedHeaders(),
			MaxAge:           ConfigFacade.GetInt("cors.max_age"),
			AllowCredentials: ConfigFacade.GetBool("cors.supports_credentials"),
		})
		if err := handler(fiberCtx.Instance()); err != nil {
			panic(err)
		}
	}
}

func allowedMethods() []string {
	allowedMethodConfigs := ConfigFacade.Get("cors.allowed_methods").([]string)
	for _, method := range allowedMethodConfigs {
		if method == "*" {
			return []string{http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch}
		}
	}
	return allowedMethodConfigs
}

func allowedOrigins() []string {
	allowedOriginConfigs := ConfigFacade.Get("cors.allowed_origins").([]string)
	for _, origin := range allowedOriginConfigs {
		if origin == "*" {
			return []string{"*"}
		}
	}
	return allowedOriginConfigs
}

func allowedHeaders() []string {
	allowedHeaderConfigs := ConfigFacade.Get("cors.allowed_headers").([]string)
	for _, header := range allowedHeaderConfigs {
		if header == "*" {
			return []string{}
		}
	}
	return allowedHeaderConfigs
}

func exposedHeaders() []string {
	exposedHeaderConfigs := ConfigFacade.Get("cors.exposed_headers").([]string)
	for _, header := range exposedHeaderConfigs {
		if header == "*" {
			return []string{}
		}
	}
	return exposedHeaderConfigs
}
