package fiber

import (
	"bytes"
	"net/http"

	"github.com/gofiber/fiber/v2"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/valyala/fasthttp"
)

type Response struct {
	instance *fiber.Ctx
	origin   httpcontract.ResponseOrigin
}

func NewResponse(instance *fiber.Ctx, origin httpcontract.ResponseOrigin) *Response {
	return &Response{instance, origin}
}

func (r *Response) Data(code int, contentType string, data []byte) {
	if err := r.instance.Status(code).Type(contentType).Send(data); err != nil {
		panic(err)
	}
}

func (r *Response) Download(filepath, filename string) {
	if err := r.instance.Download(filepath, filename); err != nil {
		panic(err)
	}
}

func (r *Response) File(filepath string) {
	if err := r.instance.SendFile(filepath); err != nil {
		panic(err)
	}
}

func (r *Response) Header(key, value string) httpcontract.Response {
	r.instance.Set(key, value)

	return r
}

func (r *Response) Json(code int, obj any) {
	if err := r.instance.Status(code).JSON(obj); err != nil {
		panic(err)
	}
}

func (r *Response) Origin() httpcontract.ResponseOrigin {
	return r.origin
}

func (r *Response) Redirect(code int, location string) {
	if err := r.instance.Redirect(location, code); err != nil {
		panic(err)
	}
}

func (r *Response) String(code int, format string, values ...any) {
	var err error
	if len(values) == 0 {
		err = r.instance.Status(code).SendString(format)
	} else {
		err = r.instance.Status(code).Type(format).SendString(values[0].(string))
	}

	if err != nil {
		panic(err)
	}
}

func (r *Response) Success() httpcontract.ResponseSuccess {
	return NewSuccess(r.instance)
}

func (r *Response) Status(code int) httpcontract.ResponseStatus {
	return NewStatus(r.instance, code)
}

func (r *Response) View() httpcontract.ResponseView {
	return NewView(r.instance)
}

func (r *Response) Writer() http.ResponseWriter {
	panic("not support")
}

func (r *Response) FastHTTPWriter() *fasthttp.Response {
	return r.instance.Response()
}

func (r *Response) Flush() {
	r.instance.Fresh()
}

type Success struct {
	instance *fiber.Ctx
}

func NewSuccess(instance *fiber.Ctx) httpcontract.ResponseSuccess {
	return &Success{instance}
}

func (r *Success) Data(contentType string, data []byte) {
	if err := r.instance.Status(http.StatusOK).Type(contentType).Send(data); err != nil {
		panic(err)
	}
}

func (r *Success) Json(obj any) {
	if err := r.instance.Status(http.StatusOK).JSON(obj); err != nil {
		panic(err)
	}
}

func (r *Success) String(format string, values ...any) {
	var err error
	if len(values) == 0 {
		err = r.instance.Status(http.StatusOK).SendString(format)
	} else {
		err = r.instance.Status(http.StatusOK).Type(format).SendString(values[0].(string))
	}

	if err != nil {
		panic(err)
	}
}

type Status struct {
	instance *fiber.Ctx
	status   int
}

func NewStatus(instance *fiber.Ctx, code int) httpcontract.ResponseSuccess {
	return &Status{instance, code}
}

func (r *Status) Data(contentType string, data []byte) {
	if err := r.instance.Status(r.status).Type(contentType).Send(data); err != nil {
		panic(err)
	}
}

func (r *Status) Json(obj any) {
	if err := r.instance.Status(r.status).JSON(obj); err != nil {
		panic(err)
	}
}

func (r *Status) String(format string, values ...any) {
	var err error
	if len(values) == 0 {
		err = r.instance.Status(r.status).SendString(format)
	} else {
		err = r.instance.Status(r.status).Type(format).SendString(values[0].(string))
	}

	if err != nil {
		panic(err)
	}
}

func ResponseMiddleware() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		switch ctx := ctx.(type) {
		case *Context:
			ctx.Instance().Response().ResetBody()
		}

		ctx.Request().Next()
	}
}

type ResponseOrigin struct {
	*fiber.Ctx
}

func (w *ResponseOrigin) Body() *bytes.Buffer {
	return bytes.NewBuffer(w.Ctx.Response().Body())
}

func (w *ResponseOrigin) Header() http.Header {
	result := http.Header{}
	w.Ctx.Response().Header.VisitAll(func(key, value []byte) {
		result.Add(string(key), string(value))
	})

	return result
}

func (w *ResponseOrigin) Size() int {
	return len(w.Ctx.Response().Body())
}

func (w *ResponseOrigin) Status() int {
	return w.Ctx.Response().StatusCode()
}
