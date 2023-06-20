package fiber

import (
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/goravel/framework/contracts/config"
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
	return func(fiberCtx *fiber.Ctx) error {
		handler(NewFiberContext(fiberCtx))
		return nil
	}
}

func middlewareToFiberHandler(handler httpcontract.Middleware) fiber.Handler {
	return func(fiberCtx *fiber.Ctx) error {
		handler(NewFiberContext(fiberCtx))
		return nil
	}
}

func getDebugLog(config config.Config) fiber.Handler {
	// TODO: add debug log

	return nil
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

func runningInConsole() bool {
	args := os.Args

	return len(args) >= 2 && args[1] == "artisan"
}
