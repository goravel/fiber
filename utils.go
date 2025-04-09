package fiber

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/route"
)

// nilHandler is a nil handler for global middleware to build chain.
var nilHandler contractshttp.HandlerFunc = func(ctx contractshttp.Context) error {
	// TODO if use a fiber middleware as global middleware by type assert, they already call ctx.Next(),
	// if duplicate call ctx.Next() will cause a error. Need to find a better way to handle this.
	if ctx.Value("no_next") != nil {
		// reset no_next flag for next global middleware
		ctx.WithValue("no_next", nil)
		return nil
	}

	fiberCtx := ctx.(*Context)
	return fiberCtx.Instance().Next()
}

// invalidFiber instance.Context() will be nil when the request is timeout,
// the request will panic if ctx.Response() is called in this situation.
func invalidFiber(instance *fiber.Ctx) bool {
	return instance.Context() == nil
}

func pathToFiberPath(relativePath string) string {
	return bracketToColon(mergeSlashForPath(relativePath))
}

func middlewaresToFiberHandlers(middlewares []contractshttp.Middleware) []fiber.Handler {
	var fiberHandlers []fiber.Handler
	for _, item := range middlewares {
		fiberHandlers = append(fiberHandlers, middlewareToFiberHandler(item))
	}

	return fiberHandlers
}

func middlewareToFiberHandler(middleware contractshttp.Middleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer func() {
			contextRequestPool.Put(context.request)
			contextResponsePool.Put(context.response)
			context.request = nil
			context.response = nil
			contextPool.Put(context)
		}()

		return middleware(nilHandler).ServeHTTP(context)
	}
}

func handlerToFiberHandler(middlewares []contractshttp.Middleware, handler contractshttp.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer func() {
			contextRequestPool.Put(context.request)
			contextResponsePool.Put(context.response)
			context.request = nil
			context.response = nil
			contextPool.Put(context)
		}()

		// build handler chain
		h := route.Chain(middlewares...).Handler(handler)

		return h.ServeHTTP(context)
	}
}

func colonToBracket(relativePath string) string {
	arr := strings.Split(relativePath, "/")
	var newArr []string
	for _, item := range arr {
		if strings.HasPrefix(item, ":") {
			item = "{" + strings.ReplaceAll(item, ":", "") + "}"
		}
		newArr = append(newArr, item)
	}

	return strings.Join(newArr, "/")
}

func bracketToColon(relativePath string) string {
	compileRegex := regexp.MustCompile(`{(.*?)}`)
	matchArr := compileRegex.FindAllStringSubmatch(relativePath, -1)

	for _, item := range matchArr {
		relativePath = strings.ReplaceAll(relativePath, item[0], ":"+item[1])
	}

	return relativePath
}

func mergeSlashForPath(path string) string {
	path = strings.ReplaceAll(path, "//", "/")

	return strings.ReplaceAll(path, "//", "/")
}
