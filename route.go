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
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	network := fiber.NetworkTCP
	prefork := config.GetBool("http.drivers.fiber.prefork", false)
	// Fiber not support prefork on dual stack
	// https://docs.gofiber.io/api/fiber#config
	if prefork {
		network = fiber.NetworkTCP4
	}

	app := fiber.New(fiber.Config{
		Prefork:               prefork,
		BodyLimit:             config.GetInt("http.drivers.fiber.body_limit", 4096) << 10,
		ReadBufferSize:        config.GetInt("http.drivers.fiber.header_limit", 4096),
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		Network:               network,
		Views:                 views,
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
	r.instance.Use(func(ctx *fiber.Ctx) error {
		if response := handler(NewContext(ctx)); response != nil {
			return response.Render()
		}

		return nil
	})
}

// GlobalMiddleware set global middleware
// GlobalMiddleware 设置全局中间件
func (r *Route) GlobalMiddleware(middlewares ...httpcontract.Middleware) {
	debug := r.config.GetBool("app.debug", false)
	timeout := time.Duration(r.config.GetInt("http.request_timeout", 3)) * time.Second
	fiberHandlers := []fiber.Handler{
		recover.New(recover.Config{
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

	globalMiddlewares := append([]httpcontract.Middleware{
		Cors(), Timeout(timeout),
	}, middlewares...)
	fiberHandlers = append(fiberHandlers, middlewaresToFiberHandlers(globalMiddlewares)...)

	r.setMiddlewares(fiberHandlers)
}

// Listen listen server
// Listen 监听服务器
func (r *Route) Listen(l net.Listener) error {
	return r.instance.Listener(l)
}

// ListenTLS listen TLS server
// ListenTLS 监听 TLS 服务器
func (r *Route) ListenTLS(l net.Listener, certFile, keyFile string) error {
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

	r.instance.SetTLSHandler(tlsHandler)
	return r.Listen(tls.NewListener(l, tlsConfig))
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

	r.outputRoutes()
	color.Green().Println(termlink.Link("[HTTP] Listening and serving HTTP on", "http://"+host[0]))

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

	r.outputRoutes()
	color.Green().Println(termlink.Link("[HTTPS] Listening and serving HTTPS on", "https://"+host))

	return r.instance.ListenTLS(host, certFile, keyFile)
}

// Stop gracefully shuts down the server
// Stop 优雅退出HTTP Server
func (r *Route) Stop(ctx ...context.Context) error {
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

func (r *Route) setMiddlewares(middlewares []fiber.Handler) {
	for _, middleware := range middlewares {
		r.instance.Use(middleware)
	}
}
