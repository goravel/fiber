package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/path"
)

var config = `map[string]any{
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

func main() {
	packages.Setup(os.Args).
		Install(
			modify.GoFile(path.Config("app.go")).
				Find(match.Imports()).Modify(modify.AddImport(packages.GetModulePath())).
				Find(match.Providers()).Modify(modify.Register("&fiber.ServiceProvider{}")),
			modify.GoFile(path.Config("http.go")).
				Find(match.Imports()).
				Modify(
					modify.AddImport("github.com/goravel/fiber/facades", "fiberfacades"), modify.AddImport("github.com/goravel/framework/support/path"),
					modify.AddImport("github.com/gofiber/template/html/v2"), modify.AddImport("github.com/gofiber/fiber/v2"),
				).
				Find(match.Config("http.drivers")).Modify(modify.AddConfig("fiber", config)),
		).
		Uninstall(
			modify.GoFile(path.Config("app.go")).
				Find(match.Imports()).Modify(modify.RemoveImport(packages.GetModulePath())).
				Find(match.Providers()).Modify(modify.Unregister("&fiber.ServiceProvider{}")),
			modify.GoFile(path.Config("http.go")).
				Find(match.Imports()).
				Modify(
					modify.RemoveImport("github.com/goravel/fiber/facades", "fiberfacades"), modify.RemoveImport("github.com/goravel/framework/support/path"),
					modify.RemoveImport("github.com/gofiber/template/html/v2"), modify.RemoveImport("github.com/gofiber/fiber/v2"),
				).
				Find(match.Config("http.drivers")).Modify(modify.RemoveConfig("fiber")),
		).
		Execute()
}
