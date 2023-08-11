package fiber

import (
	"bytes"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	httpcontract "github.com/goravel/framework/contracts/http"
)

type FiberResponse struct {
	instance *fiber.Ctx
	origin   httpcontract.ResponseOrigin
}

func NewFiberResponse(instance *fiber.Ctx, origin httpcontract.ResponseOrigin) *FiberResponse {
	return &FiberResponse{instance, origin}
}

func (r *FiberResponse) Data(code int, contentType string, data []byte) {
	_ = r.instance.Status(code).Type(contentType).Send(data)
}

func (r *FiberResponse) Download(filepath, filename string) {
	_ = r.instance.Download(filepath, filename)
}

func (r *FiberResponse) File(filepath string) {
	_ = r.instance.SendFile(filepath)
}

func (r *FiberResponse) Header(key, value string) httpcontract.Response {
	r.instance.Set(key, value)

	return r
}

func (r *FiberResponse) Json(code int, obj any) {
	_ = r.instance.Status(code).JSON(obj)
}

func (r *FiberResponse) Origin() httpcontract.ResponseOrigin {
	return r.origin
}

func (r *FiberResponse) Redirect(code int, location string) {
	_ = r.instance.Redirect(location, code)
}

func (r *FiberResponse) String(code int, format string, values ...any) {
	if len(values) == 0 {
		_ = r.instance.Status(code).Type(format).SendString(format)
		return
	}

	_ = r.instance.Status(code).Type(format).SendString(values[0].(string))
}

func (r *FiberResponse) Success() httpcontract.ResponseSuccess {
	return NewFiberSuccess(r.instance)
}

func (r *FiberResponse) Status(code int) httpcontract.ResponseStatus {
	return NewFiberStatus(r.instance, code)
}

func (r *FiberResponse) Writer() http.ResponseWriter {
	// Fiber doesn't support this
	return nil
}

func (r *FiberResponse) Flush() {
	r.instance.Fresh()
}

type FiberSuccess struct {
	instance *fiber.Ctx
}

func NewFiberSuccess(instance *fiber.Ctx) httpcontract.ResponseSuccess {
	return &FiberSuccess{instance}
}

func (r *FiberSuccess) Data(contentType string, data []byte) {
	_ = r.instance.Type(contentType).Send(data)
}

func (r *FiberSuccess) Json(obj any) {
	_ = r.instance.Status(http.StatusOK).JSON(obj)
}

func (r *FiberSuccess) String(format string, values ...any) {
	if len(values) == 0 {
		_ = r.instance.Status(http.StatusOK).Type(format).SendString(format)
		return
	}

	_ = r.instance.Status(http.StatusOK).Type(format).SendString(values[0].(string))
}

type FiberStatus struct {
	instance *fiber.Ctx
	status   int
}

func NewFiberStatus(instance *fiber.Ctx, code int) httpcontract.ResponseSuccess {
	return &FiberStatus{instance, code}
}

func (r *FiberStatus) Data(contentType string, data []byte) {
	_ = r.instance.Status(r.status).Type(contentType).Send(data)
}

func (r *FiberStatus) Json(obj any) {
	_ = r.instance.Status(r.status).JSON(obj)
}

func (r *FiberStatus) String(format string, values ...any) {
	if len(values) == 0 {
		_ = r.instance.Status(http.StatusOK).Type(format).SendString(format)
		return
	}

	_ = r.instance.Status(http.StatusOK).Type(format).SendString(values[0].(string))
}

func FiberResponseMiddleware() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		o := &ResponseOrigin{}
		switch ctx := ctx.(type) {
		case *FiberContext:
			ctx.Instance().Response().Reset()
		}

		ctx.WithValue("responseOrigin", o)
		ctx.Request().Next()
	}
}

type ResponseOrigin struct {
	*fasthttp.Response
}

func (w *ResponseOrigin) Body() *bytes.Buffer {
	return bytes.NewBuffer(w.Response.Body())
}

func (w *ResponseOrigin) Header() http.Header {
	result := http.Header{}
	w.Response.Header.VisitAll(func(key, value []byte) {
		result.Add(string(key), string(value))
	})

	return result
}

func (w *ResponseOrigin) Size() int {
	return len(w.Response.Body())
}

func (w *ResponseOrigin) Status() int {
	return w.Response.StatusCode()
}
