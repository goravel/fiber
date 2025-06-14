package fiber

import contractsroute "github.com/goravel/framework/contracts/route"

type Action struct {
	path string
}

func NewAction(method, path string) contractsroute.Action {
	routes[path] = contractsroute.Info{
		Method: method,
		Path:   path,
	}

	return &Action{
		path: path,
	}
}

func (r *Action) Name(name string) contractsroute.Action {
	info := routes[r.path]
	info.Name = name
	routes[r.path] = info

	return r
}
