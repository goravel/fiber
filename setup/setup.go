package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/env"
	"github.com/goravel/framework/support/path"
)

func main() {
	setup := packages.Setup(os.Args)
	config := `map[string]any{
        // immutable mode, see https://docs.gofiber.io/#zero-allocation
        // WARNING: This option is dangerous. Only change it if you fully understand the potential consequences.
        "immutable": true,
        // prefork mode, see https://docs.gofiber.io/api/fiber/#config
        "prefork": false,
        // Optional, default is 4096 KB
        "body_limit": 4096,
        "header_limit": 4096,
        "route": func() (route.Route, error) {
            return fiberfacades.Route("fiber"), nil
        },
        // Optional, default is "html/template"
        "template": func() (fiber.Views, error) {
            return html.New(path.Resource("views"), ".tmpl"), nil
        },
    }`
	moduleImport := setup.Paths().Module().Import()
	fiberServiceProvider := "&fiber.ServiceProvider{}"
	appConfigPath := path.Config("app.go")
	httpConfigPath := path.Config("http.go")
	routeContract := "github.com/goravel/framework/contracts/route"
	fiberFacade := "github.com/goravel/fiber/facades"
	html := "github.com/gofiber/template/html/v2"
	supportPath := "github.com/goravel/framework/support/path"
	fiber := "github.com/gofiber/fiber/v2"
	httpDriversConfig := match.Config("http.drivers")
	httpConfig := match.Config("http")

	setup.Install(
		// Add fiber service provider to app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Imports()).Modify(modify.AddImport(moduleImport)).
			Find(match.Providers()).Modify(modify.Register(fiberServiceProvider))),

		// Add fiber service provider to providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.AddProviderApply(moduleImport, fiberServiceProvider)),

		// Add fiber config to http.go
		modify.GoFile(httpConfigPath).
			Find(match.Imports()).
			Modify(
				modify.AddImport(routeContract),
				modify.AddImport(fiberFacade, "fiberfacades"), modify.AddImport(supportPath),
				modify.AddImport(html), modify.AddImport(fiber),
			).
			Find(httpDriversConfig).Modify(modify.AddConfig("fiber", config)).
			Find(httpConfig).Modify(modify.AddConfig("default", `"fiber"`)),
	).Uninstall(
		// Remove fiber config from http.go
		modify.GoFile(httpConfigPath).
			Find(httpDriversConfig).Modify(modify.RemoveConfig("fiber")).
			Find(httpConfig).Modify(modify.AddConfig("default", `""`)).
			Find(match.Imports()).
			Modify(
				modify.RemoveImport(routeContract),
				modify.RemoveImport(fiberFacade, "fiberfacades"), modify.RemoveImport(supportPath),
				modify.RemoveImport(html), modify.RemoveImport(fiber),
			),

		// Remove fiber service provider from app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Providers()).Modify(modify.Unregister(fiberServiceProvider)).
			Find(match.Imports()).Modify(modify.RemoveImport(moduleImport))),

		// Remove fiber service provider from providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.RemoveProviderApply(moduleImport, fiberServiceProvider)),
	).Execute()
}
