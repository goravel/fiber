package fiber

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gookit/color"

	"github.com/goravel/framework/contracts/config"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

// FiberRoute fiber route
// FiberRoute 光纤路由
type FiberRoute struct {
	route.Route
	config   config.Config
	instance *fiber.App
}

// NewFiberRoute create new fiber route instance
// NewFiberRoute 创建新的光纤路由实例
func NewFiberRoute(config config.Config) *FiberRoute {
	app := fiber.New(fiber.Config{
		EnableIPValidation: true,
		EnablePrintRoutes:  true,
		StreamRequestBody:  true,
		ServerHeader:       "Goravel",
		JSONEncoder:        sonic.Marshal,
		JSONDecoder:        sonic.Unmarshal,
	})
	app.Use(recover.New())
	return &FiberRoute{
		config:   config,
		instance: app,
	}
}

// Fallback set fallback handler
// Fallback 设置回退处理程序
func (r *FiberRoute) Fallback(handler httpcontract.HandlerFunc) {
	r.instance.Use(handlerToFiberHandler(handler))
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *FiberRoute) GlobalMiddleware(middlewares ...httpcontract.Middleware) {
	if len(middlewares) > 0 {
		r.instance.Use(middlewaresToFiberHandlers(middlewares)...)
	}
	r.Route = NewFiberGroup(
		r.instance,
		"",
		[]httpcontract.Middleware{},
		[]httpcontract.Middleware{FiberResponseMiddleware()},
	)
}

// Run run server
// Run 运行服务器
func (r *FiberRoute) Run(host ...string) error {
	if len(host) == 0 {
		defaultHost := r.config.GetString("http.host")
		if defaultHost == "" {
			return errors.New("host can't be empty")
		}

		defaultPort := r.config.GetString("http.port")
		if defaultPort == "" {
			return errors.New("port can't be empty")
		}
		completeHost := defaultHost + ":" + defaultPort
		host = append(host, completeHost)
	}

	r.outputRoutes()
	color.Greenln("[HTTP] Listening and serving HTTP on " + host[0])

	return r.instance.Listen(host[0])
}

// RunTLS run TLS server
// RunTLS 运行 TLS 服务器
func (r *FiberRoute) RunTLS(host ...string) error {
	if len(host) == 0 {
		defaultHost := r.config.GetString("http.tls.host")
		if defaultHost == "" {
			return errors.New("host can't be empty")
		}

		defaultPort := r.config.GetString("http.tls.port")
		if defaultPort == "" {
			return errors.New("port can't be empty")
		}
		completeHost := defaultHost + ":" + defaultPort
		host = append(host, completeHost)
	}

	certFile := r.config.GetString("http.tls.ssl.cert")
	keyFile := r.config.GetString("http.tls.ssl.key")

	return r.RunTLSWithCert(host[0], certFile, keyFile)
}

// RunTLSWithCert run TLS server with cert file and key file
// RunTLSWithCert 使用证书文件和密钥文件运行 TLS 服务器
func (r *FiberRoute) RunTLSWithCert(host, certFile, keyFile string) error {
	if host == "" {
		return errors.New("host can't be empty")
	}
	if certFile == "" || keyFile == "" {
		return errors.New("certificate can't be empty")
	}

	r.outputRoutes()
	color.Greenln("[HTTPS] Listening and serving HTTPS on " + host)

	return r.instance.ListenTLS(host, certFile, keyFile)
}

// ServeHTTP implement http.Handler interface
// ServeHTTP 实现 http.Handler 接口
func (r *FiberRoute) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// TODO: implement
}

// outputRoutes output all routes
// outputRoutes 输出所有路由
func (r *FiberRoute) outputRoutes() {
	if r.config.GetBool("app.debug") && !runningInConsole() {
		for _, item := range r.instance.GetRoutes() {
			fmt.Printf("%-10s %s\n", item.Method, colonToBracket(item.Path))
		}
	}
}
