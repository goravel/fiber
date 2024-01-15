package fiber

import (
	"bytes"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	contractshttp "github.com/goravel/framework/contracts/http"
)

type ContextResponse struct {
	instance *fiber.Ctx
	origin   contractshttp.ResponseOrigin
}

func NewContextResponse(instance *fiber.Ctx, origin contractshttp.ResponseOrigin) *ContextResponse {
	return &ContextResponse{instance, origin}
}

func (r *ContextResponse) Cookie(cookie contractshttp.Cookie) contractshttp.ContextResponse {
	r.instance.Cookie(&fiber.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Expires:  cookie.Expires,
		Secure:   cookie.Secure,
		HTTPOnly: cookie.HttpOnly,
		MaxAge:   cookie.MaxAge,
		SameSite: cookie.SameSite,
	})

	return r
}

func (r *ContextResponse) Data(code int, contentType string, data []byte) contractshttp.Response {
	return &DataResponse{code, contentType, data, r.instance}
}

func (r *ContextResponse) Download(filepath, filename string) contractshttp.Response {
	return &DownloadResponse{filename, filepath, r.instance}
}

func (r *ContextResponse) File(filepath string) contractshttp.Response {
	return &FileResponse{filepath, r.instance}
}

func (r *ContextResponse) Header(key, value string) contractshttp.ContextResponse {
	r.instance.Set(key, value)

	return r
}

func (r *ContextResponse) Json(code int, obj any) contractshttp.Response {
	return &JsonResponse{code, obj, r.instance}
}

func (r *ContextResponse) Origin() contractshttp.ResponseOrigin {
	return r.origin
}

func (r *ContextResponse) Redirect(code int, location string) contractshttp.Response {
	return &RedirectResponse{code, location, r.instance}
}

func (r *ContextResponse) String(code int, format string, values ...any) contractshttp.Response {
	return &StringResponse{code, format, r.instance, values}
}

func (r *ContextResponse) Success() contractshttp.ResponseSuccess {
	return NewSuccess(r.instance)
}

func (r *ContextResponse) Status(code int) contractshttp.ResponseStatus {
	return NewStatus(r.instance, code)
}

func (r *ContextResponse) View() contractshttp.ResponseView {
	return NewView(r.instance)
}

func (r *ContextResponse) Flush() {
	r.instance.Fresh()
}

func (r *ContextResponse) WithoutCookie(name string) contractshttp.ContextResponse {
	r.instance.Cookie(&fiber.Cookie{
		Name:   name,
		MaxAge: -1,
	})

	return r
}

func (r *ContextResponse) Writer() http.ResponseWriter {
	return &WriterAdapter{r.instance}
}

type WriterAdapter struct {
	instance *fiber.Ctx
}

func (w *WriterAdapter) Header() http.Header {
	result := http.Header{}
	w.instance.Request().Header.VisitAll(func(key, value []byte) {
		result.Add(utils.UnsafeString(key), utils.UnsafeString(value))
	})

	return result
}

func (w *WriterAdapter) Write(data []byte) (int, error) {
	return w.instance.Context().Write(data)
}

func (w *WriterAdapter) WriteHeader(code int) {
	w.instance.Context().SetStatusCode(code)
}

type Success struct {
	instance *fiber.Ctx
}

func NewSuccess(instance *fiber.Ctx) contractshttp.ResponseSuccess {
	return &Success{instance}
}

func (r *Success) Data(contentType string, data []byte) contractshttp.Response {
	return &DataResponse{http.StatusOK, contentType, data, r.instance}
}

func (r *Success) Json(obj any) contractshttp.Response {
	return &JsonResponse{http.StatusOK, obj, r.instance}
}

func (r *Success) String(format string, values ...any) contractshttp.Response {
	return &StringResponse{http.StatusOK, format, r.instance, values}
}

type Status struct {
	instance *fiber.Ctx
	status   int
}

func NewStatus(instance *fiber.Ctx, code int) contractshttp.ResponseSuccess {
	return &Status{instance, code}
}

func (r *Status) Data(contentType string, data []byte) contractshttp.Response {
	return &DataResponse{r.status, contentType, data, r.instance}
}

func (r *Status) Json(obj any) contractshttp.Response {
	return &JsonResponse{r.status, obj, r.instance}
}

func (r *Status) String(format string, values ...any) contractshttp.Response {
	return &StringResponse{r.status, format, r.instance, values}
}

func ResponseMiddleware() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
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
		result.Add(utils.UnsafeString(key), utils.UnsafeString(value))
	})

	return result
}

func (w *ResponseOrigin) Size() int {
	return len(w.Ctx.Response().Body())
}

func (w *ResponseOrigin) Status() int {
	return w.Ctx.Response().StatusCode()
}
