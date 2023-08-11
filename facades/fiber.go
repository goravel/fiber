package facades

import (
	"log"

	"github.com/goravel/fiber"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
)

func Http() http.Context {
	instance, err := fiber.App.Make(fiber.HttpBinding)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*fiber.Context)
}

func Route() route.Engine {
	instance, err := fiber.App.Make(fiber.RouteBinding)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*fiber.Route)
}
