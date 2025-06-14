package fiber

import contractsroute "github.com/goravel/framework/contracts/route"

type Action struct {
	method string
	path   string
}

func NewAction(method, path string) contractsroute.Action {
	if _, ok := routes[path]; !ok {
		routes[path] = make(map[string]contractsroute.Info)
	}

	routes[path][method] = contractsroute.Info{
		Method: method,
		Path:   path,
	}

	return &Action{
		method: method,
		path:   path,
	}
}

func (r *Action) Name(name string) contractsroute.Action {
	info := routes[r.path][r.method]
	info.Name = name
	routes[r.path][r.method] = info

	return r
}
