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

func middlewaresToFiberHandlers(middlewares []httpcontract.Middleware, prefix string) []any {
	var fiberHandlers []any
	for _, item := range middlewares {
		fiberHandlers = append(fiberHandlers, middlewareToFiberHandler(item, prefix))
	}

	return fiberHandlers
}

func handlerToFiberHandler(handler httpcontract.HandlerFunc) fiber.Handler {
	return func(fiberCtx *fiber.Ctx) error {
		handler(NewContext(fiberCtx))
		return nil
	}
}

func middlewareToFiberHandler(handler httpcontract.Middleware, prefix string) fiber.Handler {
	return func(fiberCtx *fiber.Ctx) error {
		prefix = strings.Split(prefix, ":")[0]
		if !strings.Contains(fiberCtx.Path(), prefix) {
			return fiberCtx.Next()
		}

		handler(NewContext(fiberCtx))
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
