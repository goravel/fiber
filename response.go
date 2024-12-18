package fiber

import (
	"bufio"
	"net/url"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	contractshttp "github.com/goravel/framework/contracts/http"
)

type DataResponse struct {
	code        int
	contentType string
	data        []byte
	instance    *fiber.Ctx
}

func (r *DataResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	r.instance.Response().Header.SetContentType(r.contentType)
	return r.instance.Status(r.code).Send(r.data)
}

type DownloadResponse struct {
	filename string
	filepath string
	instance *fiber.Ctx
}

func (r *DownloadResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Download(r.filepath, r.filename)
}

type FileResponse struct {
	filepath string
	instance *fiber.Ctx
}

func (r *FileResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	dir, file := filepath.Split(r.filepath)
	escapedFile := url.PathEscape(file)
	escapedPath := filepath.Join(dir, escapedFile)

	return r.instance.SendFile(escapedPath, true)
}

type JsonResponse struct {
	code     int
	obj      any
	instance *fiber.Ctx
}

func (r *JsonResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Status(r.code).JSON(r.obj)
}

func (r *JsonResponse) Abort() error {
	return nil
}

type NoContentResponse struct {
	code     int
	instance *fiber.Ctx
}

func (r *NoContentResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Status(r.code).Send(nil)
}

type RedirectResponse struct {
	code     int
	location string
	instance *fiber.Ctx
}

func (r *RedirectResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Redirect(r.location, r.code)
}

type StringResponse struct {
	code     int
	format   string
	instance *fiber.Ctx
	values   []any
}

func (r *StringResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	if len(r.values) == 0 {
		return r.instance.Status(r.code).SendString(r.format)
	}

	r.instance.Response().Header.SetContentType(r.format)
	return r.instance.Status(r.code).SendString(r.values[0].(string))
}

func (r *StringResponse) Abort() error {
	return nil
}

type HtmlResponse struct {
	data     any
	instance *fiber.Ctx
	view     string
}

func (r *HtmlResponse) Render() error {
	if invalidFiber(r.instance) {
		return nil
	}

	return r.instance.Render(r.view, r.data)
}

type StreamResponse struct {
	code     int
	instance *fiber.Ctx
	writer   func(w contractshttp.StreamWriter) error
}

func (r *StreamResponse) Render() (err error) {
	if invalidFiber(r.instance) {
		return nil
	}

	r.instance.Status(r.code)

	ctx := r.instance.Context()
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		err = r.writer(w)
	})

	return err
}

// invalidFiber instance.Context() will be nil when the request is timeout,
// the request will panic if ctx.Response() is called in this situation.
func invalidFiber(instance *fiber.Ctx) bool {
	return instance.Context() == nil
}
