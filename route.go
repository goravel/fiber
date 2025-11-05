package fiber

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
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
	"github.com/spf13/cast"
)

// map[path]map[method]info
var routes = make(map[string]map[string]contractshttp.Info)

var globalRecoverCallback func(ctx contractshttp.Context, err any) = defaultRecoverCallback

// Route fiber route
// Route fiber 路由
type Route struct {
	route.Router
	config           config.Config
	driver           string
	fallback         contractshttp.HandlerFunc
	globalMiddleware []contractshttp.Middleware
	instance         *fiber.App
}

// NewRoute creates new fiber route instance
// NewRoute 创建新的 fiber 路由实例
func NewRoute(config config.Config, parameters map[string]any) (*Route, error) {
	driver := cast.ToString(parameters["driver"])
	if driver == "" {
		return nil, errors.New("please set the driver")
	}

	timeout := time.Duration(config.GetInt("http.request_timeout", 3)) * time.Second
	globalMiddleware := []contractshttp.Middleware{Timeout(timeout), Cors()}

	route := &Route{
		config:           config,
		driver:           driver,
		globalMiddleware: globalMiddleware,
	}
	if err := route.init(globalMiddleware); err != nil {
		return nil, err
	}

	return route, nil
}

// Fallback set fallback handler
// Fallback 设置回退处理程序
func (r *Route) Fallback(handler contractshttp.HandlerFunc) {
	r.fallback = handler
}

// GetGlobalMiddleware gets global middleware
func (r *Route) GetGlobalMiddleware() []contractshttp.Middleware {
	return r.globalMiddleware
}

// GetRoutes get all routes
func (r *Route) GetRoutes() []contractshttp.Info {
	paths := []string{}
	for path := range routes {
		paths = append(paths, path)
	}

	sort.Strings(paths)
	methods := []string{contractshttp.MethodGet + "|" + contractshttp.MethodHead, contractshttp.MethodHead, contractshttp.MethodGet, contractshttp.MethodPost, contractshttp.MethodPut, contractshttp.MethodDelete, contractshttp.MethodPatch, contractshttp.MethodOptions, contractshttp.MethodAny, contractshttp.MethodResource, contractshttp.MethodStatic, contractshttp.MethodStaticFile, contractshttp.MethodStaticFS}

	var infos []contractshttp.Info
	for _, path := range paths {
		for _, method := range methods {
			if info, ok := routes[path][method]; ok {
				infos = append(infos, info)
			}
		}
	}

	return infos
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *Route) GlobalMiddleware(middleware ...contractshttp.Middleware) {
	r.globalMiddleware = append(r.globalMiddleware, middleware...)
	r.init(r.globalMiddleware)
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

func (r *Route) Info(name string) contractshttp.Info {
	routes := r.GetRoutes()

	for _, route := range routes {
		if route.Name == name {
			return route
		}
	}

	return contractshttp.Info{}
}

func (r *Route) Recover(callback func(ctx contractshttp.Context, err any)) {
	globalRecoverCallback = callback
	if err := r.init(r.globalMiddleware); err != nil {
		panic(err)
	}
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

// SetGlobalMiddleware sets global middleware
func (r *Route) SetGlobalMiddleware(middlewares []contractshttp.Middleware) {
	r.globalMiddleware = middlewares
	if err := r.init(r.globalMiddleware); err != nil {
		panic(err)
	}
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

func (r *Route) init(globalMiddleware []contractshttp.Middleware) error {
	var views fiber.Views
	template, ok := r.config.Get("http.drivers." + r.driver + ".template").(fiber.Views)
	if ok {
		views = template
	} else {
		templateCallback, ok := r.config.Get("http.drivers." + r.driver + ".template").(func() (fiber.Views, error))
		if ok {
			template, err := templateCallback()
			if err != nil {
				return err
			}

			views = template
		}
	}

	dir := path.Resource("views")
	if views == nil && file.Exists(dir) {
		views = html.New(dir, ".tmpl")
	}

	immutable := r.config.GetBool(fmt.Sprintf("http.drivers.%s.immutable", r.driver), true)
	network := fiber.NetworkTCP
	prefork := r.config.GetBool(fmt.Sprintf("http.drivers.%s.prefork", r.driver), false)

	// Fiber not support prefork on dual stack
	// https://docs.gofiber.io/api/fiber#config
	if prefork {
		network = fiber.NetworkTCP4
	}

	var trustedProxies []string
	if trustedProxiesConfig, ok := r.config.Get(fmt.Sprintf("http.drivers.%s.trusted_proxies", r.driver)).([]string); ok {
		trustedProxies = trustedProxiesConfig
	}

	instance := fiber.New(fiber.Config{
		Immutable:               immutable,
		Prefork:                 prefork,
		BodyLimit:               r.config.GetInt(fmt.Sprintf("http.drivers.%s.body_limit", r.driver), 4096) << 10,
		ReadBufferSize:          r.config.GetInt(fmt.Sprintf("http.drivers.%s.header_limit", r.driver), 4096),
		DisableStartupMessage:   true,
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		Network:                 network,
		Views:                   views,
		ProxyHeader:             r.config.GetString(fmt.Sprintf("http.drivers.%s.proxy_header", r.driver), ""),
		EnableTrustedProxyCheck: r.config.GetBool(fmt.Sprintf("http.drivers.%s.enable_trusted_proxy_check", r.driver), false),
		TrustedProxies:          trustedProxies,
	})

	debug := r.config.GetBool("app.debug", false)
	handlers := []fiber.Handler{
		fiberrecover.New(fiberrecover.Config{
			EnableStackTrace: debug,
		}),
	}

	if debug {
		handlers = append(handlers, logger.New(logger.Config{
			Format:     "[HTTP] ${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
			TimeZone:   r.config.GetString("app.timezone", "UTC"),
			TimeFormat: "2006/01/02 - 15:04:05",
		}))
	}

	recoverMiddleware := func(ctx contractshttp.Context) {
		defer func() {
			if err := recover(); err != nil {
				globalRecoverCallback(ctx, err)
			}
		}()
		ctx.Request().Next()
	}
	globalMiddleware = append([]contractshttp.Middleware{recoverMiddleware}, globalMiddleware...)
	handlers = append(handlers, middlewaresToFiberHandlers(globalMiddleware)...)

	for _, handler := range handlers {
		instance.Use(handler)
	}

	r.Router = NewGroup(
		r.config,
		instance,
		"",
		[]contractshttp.Middleware{},
		[]contractshttp.Middleware{},
	)
	r.instance = instance

	return nil
}

// outputRoutes output all routes
// outputRoutes 输出所有路由
func (r *Route) outputRoutes() {
	if r.config.GetBool("app.debug") && support.RuntimeMode != support.RuntimeArtisan && support.RuntimeMode != support.RuntimeTest {
		if err := App.MakeArtisan().Call("route:list"); err != nil {
			color.Errorln(fmt.Errorf("print route list failed: %w", err))
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

func defaultRecoverCallback(ctx contractshttp.Context, err any) {
	LogFacade.WithContext(ctx).Request(ctx.Request()).Error(err)
	ctx.Request().Abort(contractshttp.StatusInternalServerError)
}
