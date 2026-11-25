package fiber

import (
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
)

type Action struct {
	method string
	// key is the route's key in the routes map: its domain and path.
	key string
	// duplicate marks a route registered again on the same domain, path and
	// method. Fiber runs the first one, so its Info is kept as is.
	duplicate bool
}

func NewAction(method, path, handler string) contractsroute.Action {
	return newAction("", method, path, handler)
}

// newAction records a route on the domain, empty for any host. The routes of
// each domain are kept apart, so the same path can be on several domains.
func newAction(domain, method, path, handler string) contractsroute.Action {
	key := routeKey(domain, path)
	if _, ok := routes[key]; !ok {
		routes[key] = make(map[string]contractshttp.Info)
	}

	if method == contractshttp.MethodGet {
		method = contractshttp.MethodGet + "|" + contractshttp.MethodHead
	}

	if _, exists := routes[key][method]; exists {
		return &Action{method: method, key: key, duplicate: true}
	}

	routes[key][method] = contractshttp.Info{
		Domain:  domain,
		Handler: handler,
		Method:  method,
		Path:    path,
	}

	return &Action{
		method: method,
		key:    key,
	}
}

func (r *Action) Name(name string) contractsroute.Action {
	if r.duplicate {
		return r
	}

	info := routes[r.key][r.method]
	info.Name = name
	routes[r.key][r.method] = info

	return r
}

func (r *Action) WithoutMiddleware(middleware ...contractshttp.Middleware) contractsroute.Action {
	if r.duplicate {
		return r
	}

	info := routes[r.key][r.method]
	info.ExcludedMiddleware = append(info.ExcludedMiddleware, middleware...)
	routes[r.key][r.method] = info

	return r
}
