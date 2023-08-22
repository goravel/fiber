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
	r.instance.All(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Get(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Post(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Delete(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Patch(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Put(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Options(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(handler, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Resource(relativePath string, controller httpcontract.ResourceController) {
	r.instance.Get(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(controller.Index, relativePath)...)
	r.instance.Post(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), r.getHandlersWithMiddlewares(controller.Store, relativePath)...)
	r.instance.Get(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath+"/{id}"), r.getHandlersWithMiddlewares(controller.Show, relativePath)...)
	r.instance.Put(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath+"/{id}"), r.getHandlersWithMiddlewares(controller.Update, relativePath)...)
	r.instance.Patch(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath+"/{id}"), r.getHandlersWithMiddlewares(controller.Update, relativePath)...)
	r.instance.Delete(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath+"/{id}"), r.getHandlersWithMiddlewares(controller.Destroy, relativePath)...)
	r.clearMiddlewares()
}

func (r *Group) Static(relativePath, root string) {
	r.instance.Static(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), root)
	r.clearMiddlewares()
}

func (r *Group) StaticFile(relativePath, filepath string) {
	r.instance.Static(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), filepath)
	r.clearMiddlewares()
}

func (r *Group) StaticFS(relativePath string, fs http.FileSystem) {
	r.instance.Use(pathToFiberPath(r.originPrefix+"/"+r.prefix+"/"+relativePath), filesystem.New(filesystem.Config{
		Root: fs,
	}))
	r.clearMiddlewares()
}

func (r *Group) getHandlersWithMiddlewares(handler httpcontract.HandlerFunc, relativePath string) []fiber.Handler {
	prefix := pathToFiberPath(r.originPrefix + "/" + r.prefix)

	fiberHandlers := []fiber.Handler{handlerToFiberHandler(handler)}
	fiberHandlers = r.addCorsMiddleware(fiberHandlers, prefix+"/"+relativePath)
	fiberHandlers = append(fiberHandlers, middlewaresToFiberHandlers(r.middlewares, pathToFiberPath(prefix))...)

	return fiberHandlers
}

func (r *Group) clearMiddlewares() {
	r.middlewares = []httpcontract.Middleware{}
}

func (r *Group) addCorsMiddleware(middlewares []fiber.Handler, fullPath string) []fiber.Handler {
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
