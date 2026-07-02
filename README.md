# Fiber

[![Doc](https://pkg.go.dev/badge/github.com/goravel/fiber)](https://pkg.go.dev/github.com/goravel/fiber)
[![Go](https://img.shields.io/github/go-mod/go-version/goravel/fiber)](https://go.dev/)
[![Release](https://img.shields.io/github/release/goravel/fiber.svg)](https://github.com/goravel/fiber/releases)
[![Test](https://github.com/goravel/fiber/actions/workflows/test.yml/badge.svg)](https://github.com/goravel/fiber/actions)
[![Report Card](https://goreportcard.com/badge/github.com/goravel/fiber)](https://goreportcard.com/report/github.com/goravel/fiber)
[![Codecov](https://codecov.io/gh/goravel/fiber/branch/master/graph/badge.svg)](https://codecov.io/gh/goravel/fiber)
![License](https://img.shields.io/github/license/goravel/fiber)

Fiber http driver for Goravel.

## Install

Run the command below in your project to install the package automatically:

```
./artisan package:install github.com/goravel/fiber
```

Or check [the setup file](./setup/setup.go) to install the package manually.

## Configuration

You can define the `template` configuration. If omitted, `DefaultTemplate()` is used automatically as a fallback, which loads views from `resources/views` and any registered package views.

You can provide a custom template configuration in two forms:

- **`func() (fiber.Views, error)`** — a callback that returns a custom template engine (e.g. to configure custom delimiters or a FuncMap).
- **`fiber.Views`** — a pre-built template engine instance.

**Custom example:**

```go
import (
    "html/template"

    "github.com/gofiber/fiber/v3"
    goravelfiber "github.com/goravel/fiber"
)

"template": func() (fiber.Views, error) {
    return goravelfiber.NewTemplate(goravelfiber.RenderOptions{
        Delims: &goravelfiber.Delims{Left: "{[", Right: "]}"},
        FuncMap: template.FuncMap{
            "upper": strings.ToUpper,
        },
    })
},
```

## Testing

Run command below to run test:

```
go test ./...
```
