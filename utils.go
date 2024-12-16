package fiber

import (
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

func handlerToFiberHandler(handler httpcontract.HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer func() {
			contextRequestPool.Put(context.request)
			contextResponsePool.Put(context.response)
			context.request = nil
			context.response = nil
			contextPool.Put(context)
		}()

		if response := handler(context); response != nil {
			return response.Render()
		}
		return nil
	}
}

func middlewareToFiberHandler(middleware httpcontract.Middleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer func() {
			contextRequestPool.Put(context.request)
			contextResponsePool.Put(context.response)
			context.request = nil
			context.response = nil
			contextPool.Put(context)
		}()

		middleware(context)
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
