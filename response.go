package fiber

import (
	"github.com/gofiber/fiber/v2"
)

type DataResponse struct {
	code        int
	contentType string
	data        []byte
	instance    *fiber.Ctx
}

func (r *DataResponse) Render() error {
	return r.instance.Status(r.code).Type(r.contentType).Send(r.data)
}

type DownloadResponse struct {
	filename string
	filepath string
	instance *fiber.Ctx
}

func (r *DownloadResponse) Render() error {
	return r.instance.Download(r.filepath, r.filename)
}

type FileResponse struct {
	filepath string
	instance *fiber.Ctx
}

func (r *FileResponse) Render() error {
	return r.instance.SendFile(r.filepath)
}

type JsonResponse struct {
	code     int
	obj      any
	instance *fiber.Ctx
}

func (r *JsonResponse) Render() error {
	return r.instance.Status(r.code).JSON(r.obj)
}

type RedirectResponse struct {
	code     int
	location string
	instance *fiber.Ctx
}

func (r *RedirectResponse) Render() error {
	return r.instance.Redirect(r.location, r.code)
}

type StringResponse struct {
	code     int
	format   string
	instance *fiber.Ctx
	values   []any
}

func (r *StringResponse) Render() error {
	if len(r.values) == 0 {
		return r.instance.Status(r.code).SendString(r.format)
	}

	return r.instance.Status(r.code).Type(r.format).SendString(r.values[0].(string))
}

type HtmlResponse struct {
	data     any
	instance *fiber.Ctx
	view     string
}

func (r *HtmlResponse) Render() error {
	return r.instance.Render(r.view, r.data)
}
