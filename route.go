package fiber

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/goravel/framework/contracts/config"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/support"
	"github.com/goravel/framework/support/color"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/json"
	"github.com/savioxavier/termlink"
)

// Route fiber route
// Route fiber 路由
type Route struct {
	route.Router
	config   config.Config
	instance *fiber.App
}

// NewRoute create new fiber route instance
// NewRoute 创建新的 fiber 路由实例
func NewRoute(config config.Config, parameters map[string]any) (*Route, error) {
	var views fiber.Views
	if driver, exist := parameters["driver"]; exist {
		template, ok := config.Get("http.drivers." + driver.(string) + ".template").(fiber.Views)
		if ok {
			views = template
		} else {
			templateCallback, ok := config.Get("http.drivers." + driver.(string) + ".template").(func() (fiber.Views, error))
			if ok {
				template, err := templateCallback()
				if err != nil {
					return nil, err
				}

				views = template
			}
		}
	}

	if views == nil && file.Exists("./resources/views") {
		views = html.New("./resources/views", ".tmpl")
	}

	app := fiber.New(fiber.Config{
		BodyLimit:      config.GetInt("http.drivers.fiber.body_limit", 4096) << 10,
		ReadBufferSize: config.GetInt("http.drivers.fiber.header_limit", 4096),
		JSONEncoder:    json.Marshal,
		JSONDecoder:    json.Unmarshal,
		Views:          views,
	})

	return &Route{
		Router: NewGroup(
			config,
			app,
			"",
			[]httpcontract.Middleware{},
			[]httpcontract.Middleware{},
		),
		config:   config,
		instance: app,
	}, nil
}

// Fallback set fallback handler
// Fallback 设置回退处理程序
func (r *Route) Fallback(handler httpcontract.HandlerFunc) {
	r.instance.Use(func(c fiber.Ctx) error {
		context := contextPool.Get().(*Context)

		context.instance = c
		if response := handler(context); response != nil {
			return response.Render()
		}

		contextRequestPool.Put(context.request)
		contextResponsePool.Put(context.response)
		context.request = nil
		context.response = nil
		contextPool.Put(context)

		return nil
	})
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *Route) GlobalMiddleware(middlewares ...httpcontract.Middleware) {
	tempMiddlewares := []any{middlewareToFiberHandler(Cors()), recover.New(recover.Config{
		EnableStackTrace: r.config.GetBool("app.debug", false),
	})}

	debug := r.config.GetBool("app.debug", false)
	if debug {
		tempMiddlewares = append(tempMiddlewares, logger.New(logger.Config{
			Format:     "[HTTP] ${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
			TimeZone:   r.config.GetString("app.timezone", "UTC"),
			TimeFormat: "2006/01/02 - 15:04:05",
		}))
	}

	for _, middleware := range middlewaresToFiberHandlers(middlewares) {
		tempMiddlewares = append(tempMiddlewares, middleware)
	}

	r.instance.Use(tempMiddlewares...)

	r.Router = NewGroup(
		r.config,
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
		defaultPort := r.config.GetString("http.port")
		if defaultPort == "" {
			return errors.New("port can't be empty")
		}
		completeHost := defaultHost + ":" + defaultPort
		host = append(host, completeHost)
	}

	network := fiber.NetworkTCP
	prefork := r.config.GetBool("http.drivers.fiber.prefork", false)
	// Fiber not support prefork on dual stack
	// https://docs.gofiber.io/api/fiber#config
	if prefork {
		network = fiber.NetworkTCP4
	}

	r.outputRoutes()
	color.Green().Println(termlink.Link("[HTTP] Listening and serving HTTP on", "http://"+host[0]))

	return r.instance.Listen(host[0], fiber.ListenConfig{DisableStartupMessage: true, EnablePrefork: prefork, ListenerNetwork: network})
}

// RunTLS run TLS server
// RunTLS 运行 TLS 服务器
func (r *Route) RunTLS(host ...string) error {
	if len(host) == 0 {
		defaultHost := r.config.GetString("http.tls.host")
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

	network := fiber.NetworkTCP
	prefork := r.config.GetBool("http.drivers.fiber.prefork", false)
	// Fiber not support prefork on dual stack
	// https://docs.gofiber.io/api/fiber#config
	if prefork {
		network = fiber.NetworkTCP4
	}

	r.outputRoutes()
	color.Green().Println(termlink.Link("[HTTPS] Listening and serving HTTPS on", "https://"+host))

	return r.instance.Listen(host, fiber.ListenConfig{DisableStartupMessage: true, EnablePrefork: prefork, ListenerNetwork: network, CertFile: certFile, CertKeyFile: keyFile})
}

// ServeHTTP serve http request (Not support)
// ServeHTTP 服务 HTTP 请求 (不支持)
func (r *Route) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	panic("not support")
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
			for _, handler := range item.Handlers {
				if strings.Contains(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), "handlerToFiberHandler") {
					fmt.Printf("%-10s %s\n", item.Method, colonToBracket(item.Path))
					break
				}
			}
		}
	}
}
