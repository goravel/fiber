package fiber

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

type FiberGroup struct {
	instance          *fiber.App
	originPrefix      string
	prefix            string
	originMiddlewares []httpcontract.Middleware
	middlewares       []httpcontract.Middleware
	lastMiddlewares   []httpcontract.Middleware
}

func NewFiberGroup(instance *fiber.App, prefix string, originMiddlewares []httpcontract.Middleware, lastMiddlewares []httpcontract.Middleware) route.Route {
	return &FiberGroup{
		instance:          instance,
		originPrefix:      prefix,
		originMiddlewares: originMiddlewares,
		lastMiddlewares:   lastMiddlewares,
	}
}

func (r *FiberGroup) Group(handler route.GroupFunc) {
	var middlewares []httpcontract.Middleware
	middlewares = append(middlewares, r.originMiddlewares...)
	middlewares = append(middlewares, r.middlewares...)
	r.middlewares = []httpcontract.Middleware{}
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)
	r.prefix = ""

	handler(NewFiberGroup(r.instance, prefix, middlewares, r.lastMiddlewares))
}

func (r *FiberGroup) Prefix(addr string) route.Route {
	r.prefix += "/" + addr

	return r
}

func (r *FiberGroup) Middleware(middlewares ...httpcontract.Middleware) route.Route {
	r.middlewares = append(r.middlewares, middlewares...)

	return r
}

func (r *FiberGroup) Any(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().All(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Delete(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Patch(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Put(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.getFiberRoutesWithMiddlewares().Options(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(handler)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Resource(relativePath string, controller httpcontract.ResourceController) {
	r.getFiberRoutesWithMiddlewares().Get(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Index)}...)
	r.getFiberRoutesWithMiddlewares().Post(pathToFiberPath(relativePath), []fiber.Handler{handlerToFiberHandler(controller.Store)}...)
	r.getFiberRoutesWithMiddlewares().Get(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Show)}...)
	r.getFiberRoutesWithMiddlewares().Put(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getFiberRoutesWithMiddlewares().Patch(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Update)}...)
	r.getFiberRoutesWithMiddlewares().Delete(pathToFiberPath(relativePath+"/{id}"), []fiber.Handler{handlerToFiberHandler(controller.Destroy)}...)
	r.clearMiddlewares()
}

func (r *FiberGroup) Static(relativePath, root string) {
	r.getFiberRoutesWithMiddlewares().Static(pathToFiberPath(relativePath), root)
	r.clearMiddlewares()
}

func (r *FiberGroup) StaticFile(relativePath, filepath string) {
	r.getFiberRoutesWithMiddlewares().Static(pathToFiberPath(relativePath), filepath)
	r.clearMiddlewares()
}

func (r *FiberGroup) StaticFS(relativePath string, fs http.FileSystem) {
	r.getFiberRoutesWithMiddlewares().Use(pathToFiberPath(relativePath), filesystem.New(filesystem.Config{
		Root: fs,
	}))
	r.clearMiddlewares()
}

func (r *FiberGroup) getFiberRoutesWithMiddlewares() fiber.Router {
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)
	r.prefix = ""
	fiberGroup := r.instance.Group(prefix)

	var middlewares []interface{}
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

func (r *FiberGroup) clearMiddlewares() {
	r.middlewares = []httpcontract.Middleware{}
}
