package fiber

import (
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
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

func NewGroup(config config.Config, instance *fiber.App, prefix string, originMiddlewares []httpcontract.Middleware, lastMiddlewares []httpcontract.Middleware) route.Router {
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

func (r *Group) Prefix(addr string) route.Router {
	r.prefix += "/" + addr

	return r
}

func (r *Group) Middleware(middlewares ...httpcontract.Middleware) route.Router {
	r.middlewares = append(r.middlewares, middlewares...)

	return r
}

func (r *Group) Any(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.All(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Get(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Post(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Delete(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Patch(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Put(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Options(r.getPath(relativePath), handlerToFiberHandler(handler), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Resource(relativePath string, controller httpcontract.ResourceController) {
	relativePath = r.getPath(relativePath)
	r.instance.Get(relativePath, handlerToFiberHandler(controller.Index), r.getMiddlewares()...)
	r.instance.Post(relativePath, handlerToFiberHandler(controller.Store), r.getMiddlewares()...)
	r.instance.Get(r.getPath(relativePath+"/{id}"), handlerToFiberHandler(controller.Show), r.getMiddlewares()...)
	r.instance.Put(r.getPath(relativePath+"/{id}"), handlerToFiberHandler(controller.Update), r.getMiddlewares()...)
	r.instance.Patch(r.getPath(relativePath+"/{id}"), handlerToFiberHandler(controller.Update), r.getMiddlewares()...)
	r.instance.Delete(r.getPath(relativePath+"/{id}"), handlerToFiberHandler(controller.Destroy), r.getMiddlewares()...)
	r.clearMiddlewares()
}

func (r *Group) Static(relativePath, root string) {
	relativePath = r.getPath(relativePath)
	r.instance.Use(r.getMiddlewaresWithPath(r.getPath(relativePath))...).Use(relativePath, static.New(root))
	r.clearMiddlewares()
}

func (r *Group) StaticFile(relativePath, filePath string) {
	relativePath = r.getPath(relativePath)
	r.instance.Use(r.getMiddlewaresWithPath(relativePath)...).Use(relativePath, func(c fiber.Ctx) error {
		dir, file := filepath.Split(filePath)
		escapedFile := url.PathEscape(file)
		escapedPath := filepath.Join(dir, escapedFile)

		return c.SendFile(escapedPath)
	})
	r.clearMiddlewares()
}

// fsWrapper is a wrapper for http.FileSystem to implement fs.FS interface
type fsWrapper struct {
	fs http.FileSystem
}

func (h fsWrapper) Open(name string) (fs.File, error) {
	return h.fs.Open(name)
}

func (r *Group) StaticFS(relativePath string, fs http.FileSystem) {
	relativePath = r.getPath(relativePath)
	r.instance.Use(r.getMiddlewaresWithPath(relativePath)...).Use(relativePath, static.New("", static.Config{
		FS: fsWrapper{fs},
	}))
	r.clearMiddlewares()
}

func (r *Group) getMiddlewares() []fiber.Handler {
	var middlewares []fiber.Handler
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.originMiddlewares)...)
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.middlewares)...)
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.lastMiddlewares)...)

	return middlewares
}

func (r *Group) getPath(relativePath string) string {
	path := pathToFiberPath(r.originPrefix + "/" + r.prefix + "/" + relativePath)
	r.prefix = ""

	return path
}

func (r *Group) getMiddlewaresWithPath(relativePath string) []any {
	var handlers []any
	handlers = append(handlers, relativePath)
	middlewares := r.getMiddlewares()

	// Fiber will panic if no middleware is provided, So we add a dummy middleware
	if len(middlewares) == 0 {
		middlewares = append(middlewares, func(c fiber.Ctx) error {
			return c.Next()
		})
	}

	for _, item := range middlewares {
		handlers = append(handlers, item)
	}

	return handlers
}

func (r *Group) clearMiddlewares() {
	r.middlewares = []httpcontract.Middleware{}
}
