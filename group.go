package fiber

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/goravel/framework/contracts/config"
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/support/debug"
	"github.com/goravel/framework/support/str"
)

type Group struct {
	config          config.Config
	instance        *fiber.App
	prefix          string
	middlewares     []contractshttp.Middleware
	lastMiddlewares []contractshttp.Middleware
}

func NewGroup(config config.Config, instance *fiber.App, prefix string, middlewares []contractshttp.Middleware, lastMiddlewares []contractshttp.Middleware) contractsroute.Router {
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

func (r *Group) Middleware(middlewares ...contractshttp.Middleware) contractsroute.Router {
	return NewGroup(r.config, r.instance, r.getFullPath(""), append(r.middlewares, middlewares...), r.lastMiddlewares)
}

func (r *Group) Any(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.All(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodAny, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Get(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Get(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodGet, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Post(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Post(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodPost, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Delete(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Delete(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodDelete, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Patch(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Patch(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodPatch, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Put(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Put(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodPut, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Options(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
	first, rest := fiberHandlerArgs(r.getMiddlewares(handler))
	r.instance.Options(r.getFiberFullPath(path), first, rest...)

	return NewAction(contractshttp.MethodOptions, r.getFullPath(path), r.getHandlerName(handler))
}

func (r *Group) Resource(path string, controller contractshttp.ResourceController) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	first, rest := fiberHandlerArgs(r.getMiddlewares(controller.Index))
	r.instance.Get(fullPath, first, rest...)
	first, rest = fiberHandlerArgs(r.getMiddlewares(controller.Store))
	r.instance.Post(fullPath, first, rest...)

	fullPathWithID := r.getFiberFullPath(path + "/{id}")
	first, rest = fiberHandlerArgs(r.getMiddlewares(controller.Show))
	r.instance.Get(fullPathWithID, first, rest...)
	first, rest = fiberHandlerArgs(r.getMiddlewares(controller.Update))
	r.instance.Put(fullPathWithID, first, rest...)
	first, rest = fiberHandlerArgs(r.getMiddlewares(controller.Update))
	r.instance.Patch(fullPathWithID, first, rest...)
	first, rest = fiberHandlerArgs(r.getMiddlewares(controller.Destroy))
	r.instance.Delete(fullPathWithID, first, rest...)

	return NewAction(contractshttp.MethodResource, r.getFullPath(path), r.getHandlerName(controller))
}

func (r *Group) Static(path, root string) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(fullPath, static.New(root, static.Config{Browse: false}))

	return NewAction(contractshttp.MethodStatic, r.getFullPath(path), r.getHandlerName(nil))
}

func (r *Group) StaticFile(path, filePath string) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(r.getMiddlewaresWithPath(fullPath, nil)...).Use(fullPath, func(c fiber.Ctx) error {
		dir, file := filepath.Split(filePath)
		escapedFile := url.PathEscape(file)
		escapedPath := filepath.Join(dir, escapedFile)

		return c.SendFile(escapedPath)
	})

	return NewAction(contractshttp.MethodStaticFile, r.getFullPath(path), r.getHandlerName(nil))
}

func (r *Group) StaticFS(path string, fileSystem http.FileSystem) contractsroute.Action {
	fullPath := r.getFiberFullPath(path)
	r.instance.Use(fullPath, static.New("", static.Config{FS: httpFSToFS{fileSystem}}))

	return NewAction(contractshttp.MethodStaticFS, r.getFullPath(path), r.getHandlerName(nil))
}

// httpFSToFS wraps an http.FileSystem to implement fs.FS for use with fiber's static middleware.
type httpFSToFS struct {
	httpFS http.FileSystem
}

func (h httpFSToFS) Open(name string) (fs.File, error) {
	if len(name) == 0 || name[0] != '/' {
		name = "/" + name
	}
	return h.httpFS.Open(name)
}

func (r *Group) getMiddlewares(handler contractshttp.HandlerFunc) []fiber.Handler {
	var middlewares []fiber.Handler
	middlewares = middlewaresToFiberHandlers(append(r.middlewares, r.lastMiddlewares...))
	if handler != nil {
		middlewares = append(middlewares, handlerToFiberHandler(handler))
	}

	return middlewares
}

func (r *Group) getMiddlewaresWithPath(path string, handler contractshttp.HandlerFunc) []any {
	var handlers []any
	handlers = append(handlers, path)
	middlewares := r.getMiddlewares(handler)

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

func (r *Group) getFiberFullPath(path string) string {
	return pathToFiberPath(r.getFullPath(path))
}

func (r *Group) getFullPath(path string) string {
	if path == "" {
		return r.prefix
	}

	return r.prefix + str.Of(path).Start("/").String()
}

func (r *Group) getHandlerName(handler any) string {
	if handler == nil {
		return ""
	}

	if res, ok := handler.(contractshttp.ResourceController); ok {
		var (
			prefix string
			t      = reflect.TypeOf(res)
		)
		if t.Kind() == reflect.Ptr {
			prefix = "*"
			t = t.Elem()
		}

		return fmt.Sprintf("%s.(%s%s)", t.PkgPath(), prefix, t.Name())
	}

	return debug.GetFuncInfo(handler).Name
}
