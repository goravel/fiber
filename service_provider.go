package fiber

import (
	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/validation"
)

const RouteBinding = "goravel.fiber.route"

var (
	App              foundation.Application
	ConfigFacade     config.Config
	LogFacade        log.Log
	ValidationFacade validation.Validation
)

type ServiceProvider struct{}

func (receiver *ServiceProvider) Register(app foundation.Application) {
	App = app

	app.Singleton(RouteBinding, func(app foundation.Application) (any, error) {
		return NewRoute(app.MakeConfig()), nil
	})
}

func (receiver *ServiceProvider) Boot(app foundation.Application) {
	ConfigFacade = app.MakeConfig()
	LogFacade = app.MakeLog()
	ValidationFacade = app.MakeValidation()

	app.Publishes("github.com/goravel/fiber", map[string]string{
		"config/cors.go": app.ConfigPath("cors.go"),
	})
}
