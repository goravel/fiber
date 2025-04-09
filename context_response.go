package fiber

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp"
)

var contextResponsePool = sync.Pool{New: func() any {
	return &ContextResponse{}
}}

type ContextResponse struct {
	instance *fiber.Ctx
	origin   contractshttp.ResponseOrigin
}

func NewContextResponse(instance *fiber.Ctx, origin contractshttp.ResponseOrigin) contractshttp.ContextResponse {
	response := contextResponsePool.Get().(*ContextResponse)
	response.instance = instance
	response.origin = origin
	return response
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

func (r *ContextResponse) Data(code int, contentType string, data []byte) error {
	if invalidFiber(r.instance) {
		return nil
	}

	r.instance.Response().Header.SetContentType(contentType)
	return r.instance.Status(code).Send(data)
}

func (r *ContextResponse) Download(filepath, filename string) error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Download(filepath, filename)
}

func (r *ContextResponse) File(fp string) error {
	if invalidFiber(r.instance) {
		return nil
	}

	dir, file := filepath.Split(fp)
	escapedFile := url.PathEscape(file)
	escapedPath := filepath.Join(dir, escapedFile)

	return r.instance.SendFile(escapedPath, true)
}

func (r *ContextResponse) Header(key, value string) contractshttp.ContextResponse {
	r.instance.Set(key, value)

	return r
}

func (r *ContextResponse) Json(code int, obj any) error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Status(code).JSON(obj)
}

func (r *ContextResponse) NoContent(code ...int) error {
	if len(code) == 0 {
		code = append(code, http.StatusNoContent)
	}

	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Status(code[0]).Send(nil)
}

func (r *ContextResponse) Origin() contractshttp.ResponseOrigin {
	return r.origin
}

func (r *ContextResponse) Redirect(code int, location string) error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Redirect(location, code)
}

func (r *ContextResponse) String(code int, format string, values ...any) error {
	if invalidFiber(r.instance) {
		return nil
	}

	if len(values) == 0 {
		return r.instance.Status(code).SendString(format)
	}

	r.instance.Response().Header.SetContentType(format)
	return r.instance.Status(code).SendString(cast.ToString(values[0]))
}

func (r *ContextResponse) Success() contractshttp.ResponseStatus {
	return NewStatus(r.instance, http.StatusOK)
}

func (r *ContextResponse) Status(code int) contractshttp.ResponseStatus {
	return NewStatus(r.instance, code)
}

func (r *ContextResponse) Stream(code int, step func(w contractshttp.StreamWriter) error) error {
	if invalidFiber(r.instance) {
		return nil
	}

	var err error
	r.instance.Status(code)
	r.instance.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err = step(w)
	})

	return err
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
	return &netHTTPResponseWriter{
		w:   r.instance.Response().BodyWriter(),
		ctx: r.instance.Context(),
	}
}

// https://github.com/valyala/fasthttp/blob/master/fasthttpadaptor/adaptor.go#L90
type netHTTPResponseWriter struct {
	statusCode int
	h          http.Header
	w          io.Writer
	ctx        *fasthttp.RequestCtx
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *netHTTPResponseWriter) Flush() {}

type wrappedConn struct {
	net.Conn

	wg   sync.WaitGroup
	once sync.Once
}

func (c *wrappedConn) Close() (err error) {
	c.once.Do(func() {
		err = c.Conn.Close()
		c.wg.Done()
	})
	return
}

func (w *netHTTPResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// Hijack assumes control of the connection, so we need to prevent fasthttp from closing it or
	// doing anything else with it.
	w.ctx.HijackSetNoResponse(true)

	conn := &wrappedConn{Conn: w.ctx.Conn()}
	conn.wg.Add(1)
	w.ctx.Hijack(func(net.Conn) {
		conn.wg.Wait()
	})

	bufW := bufio.NewWriter(conn)

	// Write any unflushed body to the hijacked connection buffer.
	unflushedBody := w.ctx.Response.Body()
	if len(unflushedBody) > 0 {
		if _, err := bufW.Write(unflushedBody); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
	}

	return conn, &bufio.ReadWriter{Reader: bufio.NewReader(conn), Writer: bufW}, nil
}

type Status struct {
	instance *fiber.Ctx
	status   int
}

func NewStatus(instance *fiber.Ctx, code int) contractshttp.ResponseStatus {
	return &Status{instance, code}
}

func (r *Status) Data(contentType string, data []byte) error {
	if invalidFiber(r.instance) {
		return nil
	}

	r.instance.Response().Header.SetContentType(contentType)
	return r.instance.Status(r.status).Send(data)
}

func (r *Status) Json(obj any) error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Status(r.status).JSON(obj)
}

func (r *Status) String(format string, values ...any) error {
	if invalidFiber(r.instance) {
		return nil
	}

	if len(values) == 0 {
		return r.instance.Status(r.status).SendString(format)
	}

	r.instance.Response().Header.SetContentType(format)
	return r.instance.Status(r.status).SendString(cast.ToString(values[0]))
}

func (r *Status) Stream(step func(w contractshttp.StreamWriter) error) error {
	if invalidFiber(r.instance) {
		return nil
	}

	var err error
	r.instance.Status(r.status)
	r.instance.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err = step(w)
	})

	return err
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
