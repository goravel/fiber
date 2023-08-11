package fiber

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gookit/color"
	"github.com/goravel/framework/support"

	"github.com/goravel/framework/contracts/config"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

// Route fiber route
// Route 光纤路由
type Route struct {
	route.Route
	config   config.Config
	instance *fiber.App
}

// NewRoute create new fiber route instance
// NewRoute 创建新的光纤路由实例
func NewRoute(config config.Config) *Route {
	app := fiber.New(fiber.Config{
		AppName:        config.GetString("app.name", "Goravel"),
		ReadBufferSize: 16384,
		// Prefork auto enable if debug mode is disabled
		Prefork:               !config.GetBool("app.debug", false),
		EnableIPValidation:    true,
		ServerHeader:          "Goravel",
		DisableStartupMessage: !config.GetBool("app.debug", false),
		JSONEncoder:           sonic.Marshal,
		JSONDecoder:           sonic.Unmarshal,
	})
	app.Use(recover.New())

	if config.GetBool("app.debug", false) {
		app.Use(logger.New(logger.Config{
			Format:     "[HTTP] ${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
			TimeZone:   config.GetString("app.timezone", "UTC"),
			TimeFormat: "2006/01/02 - 15:04:05",
		}))
	}

	return &Route{
		Route: NewGroup(app,
			"",
			[]httpcontract.Middleware{},
			[]httpcontract.Middleware{ResponseMiddleware()},
		),
		config:   config,
		instance: app,
	}
}

// Fallback set fallback handler
// Fallback 设置回退处理程序
func (r *Route) Fallback(handler httpcontract.HandlerFunc) {
	r.instance.Use(handlerToFiberHandler(handler))
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *Route) GlobalMiddleware(middlewares ...httpcontract.Middleware) {
	if len(middlewares) > 0 {
		r.instance.Use(middlewaresToFiberHandlers(middlewares)...)
	}
	r.Route = NewGroup(
		r.instance,
		"",
		[]httpcontract.Middleware{},
		[]httpcontract.Middleware{ResponseMiddleware()},
	)
}

// Run run server
// Run 运行服务器
func (r *Route) Run(host ...string) error {
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
func (r *Route) RunTLS(host ...string) error {
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
func (r *Route) RunTLSWithCert(host, certFile, keyFile string) error {
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

// ServeHTTP serve http request (Not support)
// ServeHTTP 服务 HTTP 请求 (不支持)
func (r *Route) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
}

// Test for unit test
// Test 用于单元测试
func (r *Route) Test(request *http.Request) (*http.Response, error) {
	return r.instance.Test(request)
}

// outputRoutes output all routes
// outputRoutes 输出所有路由
func (r *Route) outputRoutes() {
	if r.config.GetBool("app.debug") && support.Env != support.EnvArtisan {
		for _, item := range r.instance.GetRoutes() {
			// filter some unnecessary methods
			if item.Method == "HEAD" || item.Method == "CONNECT" || item.Method == "TRACE" {
				continue
			}
			fmt.Printf("%-10s %s\n", item.Method, colonToBracket(item.Path))
		}
	}
}
