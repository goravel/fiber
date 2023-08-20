package fiber

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/goravel/framework/contracts/config"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

type Group struct {
	config            config.Config
	instance          *fiber.App
	originPrefix      string
	prefix            string
	originMiddlewares []httpcontract.Middleware
	middlewares       []httpcontract.Middleware
	lastMiddlewares   []httpcontract.Middleware
}

func NewGroup(config config.Config, instance *fiber.App, prefix string, originMiddlewares []httpcontract.Middleware, lastMiddlewares []httpcontract.Middleware) route.Route {
	return &Group{
		config:            config,
		instance:          instance,
		originPrefix:      prefix,
		originMiddlewares: originMiddlewares,
		lastMiddlewares:   lastMiddlewares,
	}
}

func (r *Group) Group(handler route.GroupFunc) {
	var middlewares []httpcontract.Middleware
	middlewares = append(middlewares, r.originMiddlewares...)
	middlewares = append(middlewares, r.middlewares...)
	r.middlewares = []httpcontract.Middleware{}
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)
	r.prefix = ""

	handler(NewGroup(r.config, r.instance, prefix, middlewares, r.lastMiddlewares))
}

func (r *Group) Prefix(addr string) route.Route {
	r.prefix += "/" + addr

	return r
}

func (r *Group) Middleware(middlewares ...httpcontract.Middleware) route.Route {
	r.middlewares = append(r.middlewares, middlewares...)

	return r
}

func (r *Group) Any(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).All(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Delete(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Patch(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Put(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares(relativePath).Options(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Resource(relativePath string, controller httpcontract.ResourceController) {
	r.getRoutesWithMiddlewares(relativePath).Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Index)}...)
	r.getRoutesWithMiddlewares(relativePath).Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Store)}...)
	r.getRoutesWithMiddlewares(relativePath).Get(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Show)}...)
	r.getRoutesWithMiddlewares(relativePath).Put(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getRoutesWithMiddlewares(relativePath).Patch(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getRoutesWithMiddlewares(relativePath).Delete(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Destroy)}...)
	r.clearMiddlewares()
}

func (r *Group) Static(relativePath, root string) {
	r.getRoutesWithMiddlewares(relativePath).Static(pathToFiberPath(relativePath), root)
	r.clearMiddlewares()
}

func (r *Group) StaticFile(relativePath, filepath string) {
	r.getRoutesWithMiddlewares(relativePath).Static(pathToFiberPath(relativePath), filepath)
	r.clearMiddlewares()
}

func (r *Group) StaticFS(relativePath string, fs http.FileSystem) {
	r.getRoutesWithMiddlewares(relativePath).Use(pathToFiberPath(relativePath), filesystem.New(filesystem.Config{
		Root: fs,
	}))
	r.clearMiddlewares()
}

func (r *Group) getRoutesWithMiddlewares(relativePath string) fiber.Router {
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)
	fullPath := pathToFiberPath(prefix + "/" + relativePath)

	r.prefix = ""
	fiberGroup := r.instance.Group(prefix)

	var middlewares []any
	fiberOriginMiddlewares := middlewaresToFiberHandlers(r.originMiddlewares, fullPath)
	fiberMiddlewares := middlewaresToFiberHandlers(r.middlewares, fullPath)
	fiberLastMiddlewares := middlewaresToFiberHandlers(r.lastMiddlewares, fullPath)
	middlewares = append(middlewares, fiberOriginMiddlewares...)
	middlewares = append(middlewares, fiberMiddlewares...)
	middlewares = append(middlewares, fiberLastMiddlewares...)
	middlewares = r.addCorsMiddleware(middlewares, fullPath)

	if len(middlewares) > 0 {
		fiberGroup = fiberGroup.Use(middlewares...)
	}

	return fiberGroup
}

func (r *Group) clearMiddlewares() {
	r.middlewares = []httpcontract.Middleware{}
}

func (r *Group) addCorsMiddleware(middlewares []any, fullPath string) []any {
	corsPaths := r.config.Get("cors.paths").([]string)
	for _, path := range corsPaths {
		path = pathToFiberPath(path)
		if strings.HasSuffix(path, "*") {
			path = strings.ReplaceAll(path, "*", "")
			if path == "" || strings.HasPrefix(strings.TrimPrefix(fullPath, "/"), strings.TrimPrefix(path, "/")) {
				middlewares = append(middlewares, middlewareToFiberHandler(Cors(), fullPath))
				break
			}
		} else {
			if strings.TrimPrefix(fullPath, "/") == strings.TrimPrefix(path, "/") {
				middlewares = append(middlewares, middlewareToFiberHandler(Cors(), fullPath))
				break
			}
		}
	}

	return middlewares
}
