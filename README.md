# Fiber

[![Doc](https://pkg.go.dev/badge/github.com/goravel/fiber)](https://pkg.go.dev/github.com/goravel/fiber)
[![Go](https://img.shields.io/github/go-mod/go-version/goravel/fiber)](https://go.dev/)
[![Release](https://img.shields.io/github/release/goravel/fiber.svg)](https://github.com/goravel/fiber/releases)
[![Test](https://github.com/goravel/fiber/actions/workflows/test.yml/badge.svg)](https://github.com/goravel/fiber/actions)
[![Report Card](https://goreportcard.com/badge/github.com/goravel/fiber)](https://goreportcard.com/report/github.com/goravel/fiber)
[![Codecov](https://codecov.io/gh/goravel/fiber/branch/master/graph/badge.svg)](https://codecov.io/gh/goravel/fiber)
![License](https://img.shields.io/github/license/goravel/fiber)

Fiber http driver for Goravel.

## Version

| goravel/fiber | goravel/framework |
|---------------|-------------------|
| v1.1.x        | v1.13.x           |

## Install

1. Add package

```
go get -u github.com/goravel/fiber
```

2. Register service provider

```
// config/app.go
import "github.com/goravel/fiber"

"providers": []foundation.ServiceProvider{
    ...
    &fiber.ServiceProvider{},
}
```

3. Add fiber config to `config/http.go` file

```
// config/http.go
import (
    fiberfacades "github.com/goravel/fiber/facades"
    "github.com/gofiber/template/html/v2"
    "github.com/gofiber/fiber/v2"
)

"default": "fiber",

"drivers": map[string]any{
    "fiber": map[string]any{
        // prefork mode, see https://docs.gofiber.io/api/fiber/#config
        "prefork": false,
        // Optional, default is 4096 KB
        "body_limit": 4096,
        "route": func() (route.Route, error) {
            return fiberfacades.Route("fiber"), nil
        },
        // Optional, default is "html/template"
        "template": func() (fiber.Views, error) {
            return html.New("./resources/views", ".tmpl"), nil
        },
    },
},
```

## Testing

Run command below to run test:

```
go test ./...
```
