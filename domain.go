package fiber

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// domainParamsKey keeps the parameters of the request's domain in the user
// values of the request, read like route parameters.
const domainParamsKey = "goravel:domain_params"

// domainMissKey marks a request the domain has no route for, so the routes of
// any host handle it.
const domainMissKey = "goravel:domain_miss"

// domainParamRegex matches the parameters of a domain, such as {account}.
var domainParamRegex = regexp.MustCompile(`{([^}]+)}`)

// domainRouter holds the routes of a domain in an app of their own, so the
// same path can be registered on several domains.
type domainRouter struct {
	domain      string
	pattern     *regexp.Regexp
	params      []string
	app         *fiber.App
	handler     fasthttp.RequestHandler
	handlerOnce sync.Once
}

func newDomainRouter(domain string, app *fiber.App) *domainRouter {
	router := &domainRouter{
		domain: strings.ToLower(domain),
		app:    app,
	}

	if matches := domainParamRegex.FindAllStringSubmatch(router.domain, -1); len(matches) > 0 {
		for _, match := range matches {
			router.params = append(router.params, match[1])
		}

		// Each parameter matches one label of the host.
		parts := domainParamRegex.Split(router.domain, -1)
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		router.pattern = regexp.MustCompile("^" + strings.Join(parts, "([^.]+)") + "$")
	}

	return router
}

// match reports whether the host belongs to the domain, with the values of its
// parameters.
func (r *domainRouter) match(host string) (map[string]string, bool) {
	if r.pattern == nil {
		return nil, host == r.domain
	}

	matches := r.pattern.FindStringSubmatch(host)
	if matches == nil {
		return nil, false
	}

	params := make(map[string]string, len(r.params))
	for i, name := range r.params {
		params[name] = matches[i+1]
	}

	return params, true
}

// serve handles the request with the routes of the domain, and reports whether
// the domain had a route for it.
func (r *domainRouter) serve(ctx *fasthttp.RequestCtx, params map[string]string) bool {
	// The handler is taken on the first request, once the routes are registered.
	r.handlerOnce.Do(func() {
		r.handler = r.app.Handler()
	})

	ctx.SetUserValue(domainParamsKey, params)
	r.handler(ctx)

	if ctx.UserValue(domainMissKey) != nil {
		ctx.RemoveUserValue(domainMissKey)
		ctx.RemoveUserValue(domainParamsKey)

		return false
	}

	return true
}

// domainErrorHandler passes a request the domain has no route for to the
// routes of any host, and handles the other errors as fiber does.
func domainErrorHandler(ctx fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && (fiberErr.Code == fiber.StatusNotFound || fiberErr.Code == fiber.StatusMethodNotAllowed) {
		ctx.RequestCtx().SetUserValue(domainMissKey, true)

		return nil
	}

	return fiber.DefaultErrorHandler(ctx, err)
}

// domainParams returns the parameters of the request's domain.
func domainParams(ctx fiber.Ctx) map[string]string {
	params, _ := ctx.RequestCtx().UserValue(domainParamsKey).(map[string]string)

	return params
}

// requestHost returns the host of the request in lower case, without its port.
func requestHost(host string) string {
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}

	return strings.ToLower(strings.TrimSuffix(host, "."))
}
