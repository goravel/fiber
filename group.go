package fiber

import (
	"net/http"
	"net/url"
	"path/filepath"

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
	r.instance.All(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Get(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Get(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Post(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Post(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Delete(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Delete(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Patch(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Patch(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Put(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Put(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Options(relativePath string, handler httpcontract.HandlerFunc) {
	r.instance.Options(r.getPath(relativePath), r.getMiddlewares(handler)...)
	r.clearMiddlewares()
}

func (r *Group) Resource(relativePath string, controller httpcontract.ResourceController) {
	r.instance.Get(r.getPath(relativePath), r.getMiddlewares(controller.Index)...)
	r.instance.Post(r.getPath(relativePath), r.getMiddlewares(controller.Store)...)
	r.instance.Get(r.getPath(relativePath+"/{id}"), r.getMiddlewares(controller.Show)...)
	r.instance.Put(r.getPath(relativePath+"/{id}"), r.getMiddlewares(controller.Update)...)
	r.instance.Patch(r.getPath(relativePath+"/{id}"), r.getMiddlewares(controller.Update)...)
	r.instance.Delete(r.getPath(relativePath+"/{id}"), r.getMiddlewares(controller.Destroy)...)
	r.clearMiddlewares()
}

func (r *Group) Static(relativePath, root string) {
	r.instance.Use(r.getMiddlewaresWithPath(relativePath, nil)...).Static(r.getPath(relativePath), root)
	r.clearMiddlewares()
}

func (r *Group) StaticFile(relativePath, filePath string) {
	r.instance.Use(r.getMiddlewaresWithPath(relativePath, nil)...).Use(r.getPath(relativePath), func(c *fiber.Ctx) error {
		dir, file := filepath.Split(filePath)
		escapedFile := url.PathEscape(file)
		escapedPath := filepath.Join(dir, escapedFile)
		return c.SendFile(escapedPath, true)
	})
	r.clearMiddlewares()
}

func (r *Group) StaticFS(relativePath string, fs http.FileSystem) {
	r.instance.Use(r.getMiddlewaresWithPath(relativePath, nil)...).Use(r.getPath(relativePath), filesystem.New(filesystem.Config{
		Root: fs,
	}))
	r.clearMiddlewares()
}

func (r *Group) getMiddlewares(handler httpcontract.HandlerFunc) []fiber.Handler {
	var middlewares []fiber.Handler
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.originMiddlewares)...)
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.middlewares)...)
	middlewares = append(middlewares, middlewaresToFiberHandlers(r.lastMiddlewares)...)
	if handler != nil {
		middlewares = append(middlewares, handlerToFiberHandler(handler))
	}

	return middlewares
}

func (r *Group) getPath(relativePath string) string {
	path := pathToFiberPath(r.originPrefix + "/" + r.prefix + "/" + relativePath)
	r.prefix = ""

	return path
}

func (r *Group) getMiddlewaresWithPath(relativePath string, handler httpcontract.HandlerFunc) []any {
	var handlers []any
	handlers = append(handlers, r.getPath(relativePath))
	middlewares := r.getMiddlewares(handler)

	// Fiber will panic if no middleware is provided, So we add a dummy middleware
	if len(middlewares) == 0 {
		middlewares = append(middlewares, func(c *fiber.Ctx) error {
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
