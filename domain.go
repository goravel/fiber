package fiber

import (
	"errors"
	"sync"

	"github.com/gofiber/fiber/v3"
	supporthttp "github.com/goravel/framework/support/http"
	"github.com/valyala/fasthttp"
)

// domainParamsKey keeps the parameters of the request's domain in the user
// values of the request, read like route parameters.
const domainParamsKey = "goravel:domain_params"

// routeDomainKey keeps the domain serving the request in the user values of
// the request, so the route's Info is found among the routes of that domain.
const routeDomainKey = "goravel:route_domain"

// domainMissKey marks a request the domain has no route for, so the routes of
// any host handle it.
const domainMissKey = "goravel:domain_miss"

// domainRouteKey marks a request a route of the domain has run for, such as
// a static route that doesn't find the file. A 404 then belongs to the
// domain, rather than going on to the routes of any host.
const domainRouteKey = "goravel:domain_route"

// domainRouter holds the routes of a domain in an app of their own, so the
// same path can be registered on several domains. Domains that differ only in
// their parameter names share it.
type domainRouter struct {
	domain      *supporthttp.Domain
	app         *fiber.App
	handler     fasthttp.RequestHandler
	handlerOnce sync.Once
}

// serve handles the request with the routes of the domain, and reports whether
// the domain had a route for it.
func (r *domainRouter) serve(ctx *fasthttp.RequestCtx) bool {
	// The handler is taken on the first request, once the routes are registered.
	r.handlerOnce.Do(func() {
		r.handler = r.app.Handler()
	})

	r.handler(ctx)

	if ctx.UserValue(domainMissKey) != nil {
		ctx.RemoveUserValue(domainMissKey)
		ctx.RemoveUserValue(domainParamsKey)
		ctx.RemoveUserValue(routeDomainKey)

		return false
	}

	return true
}

// domainHandler runs first on every route of the domain: it records the domain
// of the route, and the values of its parameters, read like route parameters
// with the route's own names.
func domainHandler(domain *supporthttp.Domain) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		requestCtx := ctx.RequestCtx()
		params, _ := domain.Match(string(requestCtx.Host()))
		requestCtx.SetUserValue(domainParamsKey, params)
		requestCtx.SetUserValue(routeDomainKey, domain.String())
		requestCtx.SetUserValue(domainRouteKey, true)

		return ctx.Next()
	}
}

// domainErrorHandler passes a request the domain has no route for to the
// routes of any host, and handles the other errors as fiber does. Once a
// route of the domain has run, a 404 is the domain's own.
func domainErrorHandler(ctx fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && (fiberErr.Code == fiber.StatusNotFound || fiberErr.Code == fiber.StatusMethodNotAllowed) &&
		ctx.RequestCtx().UserValue(domainRouteKey) == nil {
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

// routeDomain returns the domain serving the request, empty for any host.
func routeDomain(ctx fiber.Ctx) string {
	domain, _ := ctx.RequestCtx().UserValue(routeDomainKey).(string)

	return domain
}

// routeKey returns the key of a route in the routes map: the routes of each
// domain are kept apart, so the same path can be on several domains.
func routeKey(domain, path string) string {
	return domain + path
}
