package fiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/validation"
)

const HttpBinding = "goravel.http"
const RouteBinding = "goravel.route"

var App foundation.Application

var (
	LogFacade        log.Log
	ValidationFacade validation.Validation
)

type ServiceProvider struct {
}

func (receiver *ServiceProvider) Register(app foundation.Application) {
	App = app

	app.Bind(HttpBinding, func(app foundation.Application) (any, error) {
		return NewContext(&fiber.Ctx{}), nil
	})
	app.Bind(RouteBinding, func(app foundation.Application) (any, error) {
		return NewRoute(app.MakeConfig()), nil
	})
}

func (receiver *ServiceProvider) Boot(app foundation.Application) {
	LogFacade = app.MakeLog()
	ValidationFacade = app.MakeValidation()
}
