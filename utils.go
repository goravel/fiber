package fiber

import (
	"reflect"
	"regexp"
	"strings"
	"unsafe"

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
		defer releaseContext(context)

		if response := handler(context); response != nil {
			return response.Render()
		}
		return nil
	}
}

func middlewareToFiberHandler(middleware httpcontract.Middleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		context := NewContext(c)
		defer releaseContext(context)

		middleware(context)
		return nil
	}
}

func cloneFiberContext(context *Context) *Context {
	instance := context.Instance()
	clone := instance.App().AcquireCtx(instance.Context())

	source := reflect.ValueOf(instance).Elem()
	target := reflect.ValueOf(clone).Elem()
	typeOfCtx := source.Type()

	for i := 0; i < source.NumField(); i++ {
		field := typeOfCtx.Field(i)
		if field.Name == "viewBindMap" {
			continue
		}

		targetField := target.Field(i)
		sourceField := source.Field(i)
		reflect.NewAt(targetField.Type(), unsafe.Pointer(targetField.UnsafeAddr())).Elem().Set(
			reflect.NewAt(sourceField.Type(), unsafe.Pointer(sourceField.UnsafeAddr())).Elem(),
		)
	}

	return NewContext(clone)
}

func releaseContext(context *Context) {
	if context.request != nil {
		contextRequestPool.Put(context.request)
	}
	if context.response != nil {
		contextResponsePool.Put(context.response)
	}
	context.request = nil
	context.response = nil
	context.instance = nil
	contextPool.Put(context)
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
