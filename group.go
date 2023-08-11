package fiber

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

type Group struct {
	instance          *fiber.App
	originPrefix      string
	prefix            string
	originMiddlewares []httpcontract.Middleware
	middlewares       []httpcontract.Middleware
	lastMiddlewares   []httpcontract.Middleware
}

func NewGroup(instance *fiber.App, prefix string, originMiddlewares []httpcontract.Middleware, lastMiddlewares []httpcontract.Middleware) route.Route {
	return &Group{
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

	handler(NewGroup(r.instance, prefix, middlewares, r.lastMiddlewares))
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
	r.getRoutesWithMiddlewares().All(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Delete(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Patch(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Put(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.getRoutesWithMiddlewares().Options(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *Group) Resource(relativePath string, controller httpcontract.ResourceController) {
	r.getRoutesWithMiddlewares().Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Index)}...)
	r.getRoutesWithMiddlewares().Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Store)}...)
	r.getRoutesWithMiddlewares().Get(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Show)}...)
	r.getRoutesWithMiddlewares().Put(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getRoutesWithMiddlewares().Patch(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getRoutesWithMiddlewares().Delete(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Destroy)}...)
	r.clearMiddlewares()
}

func (r *Group) Static(relativePath, root string) {
	r.getRoutesWithMiddlewares().Static(pathToFiberPath(relativePath), root)
	r.clearMiddlewares()
}

func (r *Group) StaticFile(relativePath, filepath string) {
	r.getRoutesWithMiddlewares().Static(pathToFiberPath(relativePath), filepath)
	r.clearMiddlewares()
}

func (r *Group) StaticFS(relativePath string, fs http.FileSystem) {
	r.getRoutesWithMiddlewares().Use(pathToFiberPath(relativePath), filesystem.New(filesystem.Config{
		Root: fs,
	}))
	r.clearMiddlewares()
}

func (r *Group) getRoutesWithMiddlewares() fiber.Router {
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)
	r.prefix = ""
	fiberGroup := r.instance.Group(prefix)

	var middlewares []any
	fiberOriginMiddlewares := middlewaresToFiberHandlers(r.originMiddlewares)
	fiberMiddlewares := middlewaresToFiberHandlers(r.middlewares)
	fiberLastMiddlewares := middlewaresToFiberHandlers(r.lastMiddlewares)
	middlewares = append(middlewares, fiberOriginMiddlewares...)
	middlewares = append(middlewares, fiberMiddlewares...)
	middlewares = append(middlewares, fiberLastMiddlewares...)

	if len(middlewares) > 0 {
		return fiberGroup.Use(middlewares...)
	} else {
		return fiberGroup
	}
}

func (r *Group) clearMiddlewares() {
	r.middlewares = []httpcontract.Middleware{}
}
