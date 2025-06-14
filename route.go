package fiber

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"

	"github.com/goravel/framework/contracts/config"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/support"
	"github.com/goravel/framework/support/color"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/json"
	"github.com/goravel/framework/support/path"
	"github.com/goravel/framework/support/str"
)

var globalRecoverCallback func(ctx contractshttp.Context, err any) = func(ctx contractshttp.Context, err any) {
	LogFacade.WithContext(ctx).Request(ctx.Request()).Error(err)
	ctx.Request().Abort(contractshttp.StatusInternalServerError)
}

// Route fiber route
// Route fiber 路由
type Route struct {
	route.Router
	config   config.Config
	instance *fiber.App
	fallback contractshttp.HandlerFunc
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

	dir := path.Resource("views")
	if views == nil && file.Exists(dir) {
		views = html.New(dir, ".tmpl")
	}

	immutable := config.GetBool("http.drivers.fiber.immutable", true)
	network := fiber.NetworkTCP
	prefork := config.GetBool("http.drivers.fiber.prefork", false)
	// Fiber not support prefork on dual stack
	// https://docs.gofiber.io/api/fiber#config
	if prefork {
		network = fiber.NetworkTCP4
	}

	var trustedProxies []string
	if trustedProxiesConfig, ok := config.Get("http.drivers.fiber.trusted_proxies").([]string); ok {
		trustedProxies = trustedProxiesConfig
	}

	app := fiber.New(fiber.Config{
		Immutable:               immutable,
		Prefork:                 prefork,
		BodyLimit:               config.GetInt("http.drivers.fiber.body_limit", 4096) << 10,
		ReadBufferSize:          config.GetInt("http.drivers.fiber.header_limit", 4096),
		DisableStartupMessage:   true,
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		Network:                 network,
		Views:                   views,
		ProxyHeader:             config.GetString("http.drivers.fiber.proxy_header", ""),
		EnableTrustedProxyCheck: config.GetBool("http.drivers.fiber.enable_trusted_proxy_check", false),
		TrustedProxies:          trustedProxies,
	})

	return &Route{
		Router: NewGroup(
			config,
			app,
			"",
			[]contractshttp.Middleware{},
			[]contractshttp.Middleware{},
		),
		config:   config,
		instance: app,
	}, nil
}

// Fallback set fallback handler
// Fallback 设置回退处理程序
func (r *Route) Fallback(handler contractshttp.HandlerFunc) {
	r.fallback = handler
}

// GetRoutes get all routes
func (r *Route) GetRoutes() []route.Info {
	var routes []route.Info
	for _, item := range r.instance.GetRoutes() {
		for _, handler := range item.Handlers {
			if strings.Contains(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), "handlerToFiberHandler") {
				routes = append(routes, route.Info{
					Method: item.Method,
					Path:   colonToBracket(item.Path),
				})
				break
			}
		}
	}

	return routes
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *Route) GlobalMiddleware(middlewares ...contractshttp.Middleware) {
	debug := r.config.GetBool("app.debug", false)
	fiberHandlers := []fiber.Handler{
		fiberrecover.New(fiberrecover.Config{
			EnableStackTrace: debug,
		}),
	}

	if debug {
		fiberHandlers = append(fiberHandlers, logger.New(logger.Config{
			Format:     "[HTTP] ${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
			TimeZone:   r.config.GetString("app.timezone", "UTC"),
			TimeFormat: "2006/01/02 - 15:04:05",
		}))
	}

	globalMiddlewares := []contractshttp.Middleware{Cors()}
	timeout := time.Duration(r.config.GetInt("http.request_timeout", 3)) * time.Second
	if timeout > 0 {
		globalMiddlewares = append(globalMiddlewares, Timeout(timeout))
	}
	globalMiddlewares = append(globalMiddlewares, middlewares...)
	fiberHandlers = append(fiberHandlers, middlewaresToFiberHandlers(globalMiddlewares)...)

	r.setMiddlewares(fiberHandlers)
}

func (r *Route) Recover(callback func(ctx contractshttp.Context, err any)) {
	globalRecoverCallback = callback
	middleware := middlewaresToFiberHandlers([]contractshttp.Middleware{
		func(ctx contractshttp.Context) {
			defer func() {
				if err := recover(); err != nil {
					callback(ctx, err)
				}
			}()
			ctx.Request().Next()
		},
	})
	r.setMiddlewares(middleware)
}

// Listen listen server
// Listen 监听服务器
func (r *Route) Listen(l net.Listener) error {
	r.registerFallback()
	r.outputRoutes()
	color.Green().Println("[HTTP] Listening on: " + str.Of(l.Addr().String()).Start("http://").String())

	return r.instance.Listener(l)
}

// ListenTLS listen TLS server
// ListenTLS 监听 TLS 服务器
func (r *Route) ListenTLS(l net.Listener) error {
	return r.ListenTLSWithCert(l, r.config.GetString("http.tls.ssl.cert"), r.config.GetString("http.tls.ssl.key"))
}

// ListenTLSWithCert listen TLS server with cert file and key file
// ListenTLSWithCert 使用证书文件和密钥文件监听 TLS 服务器
func (r *Route) ListenTLSWithCert(l net.Listener, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	tlsHandler := &fiber.TLSHandler{}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			cert,
		},
		GetCertificate: tlsHandler.GetClientInfo,
	}

	r.registerFallback()
	r.outputRoutes()
	color.Green().Println("[HTTPS] Listening on: " + str.Of(l.Addr().String()).Start("https://").String())

	r.instance.SetTLSHandler(tlsHandler)

	return r.instance.Listener(tls.NewListener(l, tlsConfig))
}

func (r *Route) Info(name string) route.Info {
	for _, info := range routes {
		if info.Name == name {
			return info
		}
	}

	return route.Info{}
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

	r.registerFallback()
	r.outputRoutes()
	color.Green().Println("[HTTP] Listening on: " + str.Of(host[0]).Start("http://").String())

	return r.instance.Listen(host[0])
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

	r.registerFallback()
	r.outputRoutes()
	color.Green().Println("[HTTPS] Listening on: " + str.Of(host).Start("https://").String())

	return r.instance.ListenTLS(host, certFile, keyFile)
}

// Shutdown gracefully shuts down the server
// Shutdown 优雅退出HTTP Server
func (r *Route) Shutdown(ctx ...context.Context) error {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	return r.instance.ShutdownWithContext(c)
}

// ServeHTTP serve http request (Not support)
// ServeHTTP 服务 HTTP 请求 (不支持)
func (r *Route) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	panic("not support")
}

// Test for unit test
// Test 用于单元测试
func (r *Route) Test(request *http.Request) (*http.Response, error) {
	r.registerFallback()

	return r.instance.Test(request, -1)
}

// outputRoutes output all routes
// outputRoutes 输出所有路由
func (r *Route) outputRoutes() {
	if r.config.GetBool("app.debug") && support.RuntimeMode != support.RuntimeArtisan {
		for _, item := range r.GetRoutes() {
			fmt.Printf("%-10s %s\n", item.Method, item.Path)
		}
	}
}

func (r *Route) registerFallback() {
	if r.fallback == nil {
		return
	}

	r.instance.Use(func(ctx *fiber.Ctx) error {
		if response := r.fallback(NewContext(ctx)); response != nil {
			return response.Render()
		}
		return nil
	})
}

func (r *Route) setMiddlewares(middlewares []fiber.Handler) {
	for _, middleware := range middlewares {
		r.instance.Use(middleware)
	}
}
