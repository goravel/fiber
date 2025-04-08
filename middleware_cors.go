package fiber

import (
	"github.com/goravel/framework/support/debug"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/rs/cors"
)

func Cors() http.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandleFunc(func(ctx http.Context) http.Response {
			debug.Dump("cors called")
			path := ctx.Request().Path()
			corsPaths, ok := ConfigFacade.Get("cors.paths").([]string)
			if !ok {
				return next.ServeHTTP(ctx)
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
				return next.ServeHTTP(ctx)
			}

			allowedMethods := ConfigFacade.Get("cors.allowed_methods").([]string)
			if len(allowedMethods) == 1 && allowedMethods[0] == "*" {
				allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch}
			}

			instance := cors.New(cors.Options{
				AllowedMethods:      allowedMethods,
				AllowedOrigins:      ConfigFacade.Get("cors.allowed_origins").([]string),
				AllowedHeaders:      ConfigFacade.Get("cors.allowed_headers").([]string),
				ExposedHeaders:      ConfigFacade.Get("cors.exposed_headers").([]string),
				MaxAge:              ConfigFacade.GetInt("cors.max_age"),
				AllowCredentials:    ConfigFacade.GetBool("cors.supports_credentials"),
				AllowPrivateNetwork: true,
			})

			instance.HandlerFunc(ctx.Response().Writer(), ctx.Request().Origin())

			if ctx.Request().Origin().Method == http.MethodOptions &&
				ctx.Request().Header("Access-Control-Request-Method") != "" {
				ctx.Request().Abort(http.StatusNoContent)
			}

			return next.ServeHTTP(ctx)
		})
	}
}
