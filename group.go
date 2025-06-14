package fiber

import (
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/goravel/framework/contracts/config"
	httpcontract "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/support/str"
)

type Group struct {
	config          config.Config
	instance        *fiber.App
	prefix          string
	middlewares     []httpcontract.Middleware
	lastMiddlewares []httpcontract.Middleware
}

func NewGroup(config config.Config, instance *fiber.App, prefix string, middlewares []httpcontract.Middleware, lastMiddlewares []httpcontract.Middleware) contractsroute.Router {
	return &Group{
		config:          config,
		instance:        instance,
		prefix:          prefix,
		middlewares:     middlewares,
		lastMiddlewares: lastMiddlewares,
	}
}

func (r *Group) Group(handler contractsroute.GroupFunc) {
	handler(NewGroup(r.config, r.instance, r.getFullPath(""), r.middlewares, r.lastMiddlewares))
}

func (r *Group) Prefix(path string) contractsroute.Router {
	return NewGroup(r.config, r.instance, r.getFullPath(path), r.middlewares, r.lastMiddlewares)
}

func (r *Group) Middleware(middlewares ...httpcontract.Middleware) contractsroute.Router {
	return NewGroup(r.config, r.instance, r.getFullPath(""), append(r.middlewares, middlewares...), r.lastMiddlewares)
}

func (r *Group) Any(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.All(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodAny, r.getFullPath(path))
}

func (r *Group) Get(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Get(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodGet, r.getFullPath(path))
}

func (r *Group) Post(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Post(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodPost, r.getFullPath(path))
}

func (r *Group) Delete(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Delete(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodDelete, r.getFullPath(path))
}

func (r *Group) Patch(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Patch(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodPatch, r.getFullPath(path))
}

func (r *Group) Put(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Put(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodPut, r.getFullPath(path))
}

func (r *Group) Options(path string, handler httpcontract.HandlerFunc) contractsroute.Action {
	r.instance.Options(r.getFiberFullPath(path), r.getMiddlewares(handler)...)

	return NewAction(MethodOptions, r.getFullPath(path))
}

func (r *Group) Resource(path string, controller httpcontract.ResourceController) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Get(fullPath, r.getMiddlewares(controller.Index)...)
	r.instance.Post(fullPath, r.getMiddlewares(controller.Store)...)

	fullPathWithID := r.getFiberFullPath(path + "/{id}")
	r.instance.Get(fullPathWithID, r.getMiddlewares(controller.Show)...)
	r.instance.Put(fullPathWithID, r.getMiddlewares(controller.Update)...)
	r.instance.Patch(fullPathWithID, r.getMiddlewares(controller.Update)...)
	r.instance.Delete(fullPathWithID, r.getMiddlewares(controller.Destroy)...)

	return NewAction(MethodResource, r.getFullPath(path))
}

func (r *Group) Static(path, root string) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(r.getMiddlewaresWithPath(fullPath, nil)...).Static(fullPath, root)

	return NewAction(MethodStatic, r.getFullPath(path))
}

func (r *Group) StaticFile(path, filePath string) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(r.getMiddlewaresWithPath(fullPath, nil)...).Use(fullPath, func(c *fiber.Ctx) error {
		dir, file := filepath.Split(filePath)
		escapedFile := url.PathEscape(file)
		escapedPath := filepath.Join(dir, escapedFile)

		return c.SendFile(escapedPath, true)
	})

	return NewAction(MethodStaticFile, r.getFullPath(path))
}

func (r *Group) StaticFS(path string, fs http.FileSystem) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(r.getMiddlewaresWithPath(fullPath, nil)...).Use(fullPath, filesystem.New(filesystem.Config{
		Root: fs,
	}))

	return NewAction(MethodStaticFS, r.getFullPath(path))
}

func (r *Group) getMiddlewares(handler httpcontract.HandlerFunc) []fiber.Handler {
	var middlewares []fiber.Handler
	middlewares = middlewaresToFiberHandlers(append(r.middlewares, r.lastMiddlewares...))
	if handler != nil {
		middlewares = append(middlewares, handlerToFiberHandler(handler))
	}

	return middlewares
}

func (r *Group) getMiddlewaresWithPath(path string, handler httpcontract.HandlerFunc) []any {
	var handlers []any
	handlers = append(handlers, path)
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

func (r *Group) getFiberFullPath(path string) string {
	return pathToFiberPath(r.getFullPath(path))
}

func (r *Group) getFullPath(path string) string {
	if path == "" {
		return r.prefix
	}

	return r.prefix + str.Of(path).Start("/").String()
}
