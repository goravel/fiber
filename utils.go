package fiber

import (
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/goravel/framework/http"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	httpcontract "github.com/goravel/framework/contracts/http"
)

func pathToFiberPath(relativePath string) string {
	return bracketToColon(mergeSlashForPath(relativePath))
}

func middlewaresToFiberHandlers(middlewares []httpcontract.Middleware) []fiber.Handler {
	var fiberHandlers []fiber.Handler
	for _, item := range middlewares {
		fiberHandlers = append(fiberHandlers, middlewareToFiberHandler(item))
	}

	return fiberHandlers
}

func middlewareToFiberHandler(middleware httpcontract.Middleware) fiber.Handler {
	return adaptor.HTTPMiddleware(http.HTTPMiddlewareToMiddleware)
}

func toFiberHandler(middlewares []httpcontract.Middleware, handler httpcontract.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer func() {
			contextRequestPool.Put(context.request)
			contextResponsePool.Put(context.response)
			context.request = nil
			context.response = nil
			contextPool.Put(context)
		}()

		h := http.Chain(middlewares...).Handler(handler)

		if response := h.ServeHTTP(context); response != nil {
			return response.Render()
		}

		return nil
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
