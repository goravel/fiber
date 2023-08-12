package facades

import (
	"log"

	"github.com/goravel/fiber"
	"github.com/goravel/framework/contracts/route"
)

func Route() route.Engine {
	instance, err := fiber.App.Make(fiber.RouteBinding)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*fiber.Route)
}
