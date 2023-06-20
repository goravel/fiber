# Fiber

Fiber http driver for Goravel.

This driver is still in development, please do not use it in production.

## Version

| goravel/fiber | goravel/framework |
|---------------|-------------------|
| v1.0.0        | v1.13.0           |

## Install

1. Add package

```
go get -u github.com/goravel/fiber
```

2. Register service provider, make sure it is registered first.

```
// config/app.go
import "github.com/goravel/fiber"

"providers": []foundation.ServiceProvider{
    &fiber.ServiceProvider{},
    ...
}
```

3. Add fiber config to `config/http.go` file

```
// config/http.go
import (
    fiberfacades "github.com/goravel/fiber/facades"
)

"driver": "fiber",

"drivers": map[string]any{
    ...
    "fiber": map[string]any{
        "http": func() (http.Context, error) {
            return fiberfacades.Http(), nil
        },
        "route": func() (route.Engine, error) {
            return fiberfacades.Route(), nil
        },
    },
}
```

## Testing

Run command below to run test:

```
go test ./...
```
