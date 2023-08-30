package facades

import (
	"log"

	"github.com/goravel/framework/contracts/route"

	"github.com/goravel/fiber"
)

func Route(driver string) route.Route {
	instance, err := fiber.App.MakeWith(fiber.RouteBinding, map[string]any{
		"driver": driver,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*fiber.Route)
}
