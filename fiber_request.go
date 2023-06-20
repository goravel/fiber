package fiber

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gookit/validate"
	"github.com/spf13/cast"

	filesystemcontract "github.com/goravel/framework/contracts/filesystem"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/log"
	validatecontract "github.com/goravel/framework/contracts/validation"
	"github.com/goravel/framework/filesystem"
	"github.com/goravel/framework/validation"
)

type FiberRequest struct {
	ctx        *FiberContext
	instance   *fiber.Ctx
	postData   map[string]any
	log        log.Log
	validation validatecontract.Validation
}

func NewFiberRequest(ctx *FiberContext, log log.Log, validation validatecontract.Validation) httpcontract.Request {
	postData, err := getPostData(ctx)
	if err != nil {
		LogFacade.Error(fmt.Sprintf("%+v", errors.Unwrap(err)))
	}

	return &FiberRequest{ctx: ctx, instance: ctx.instance, postData: postData, log: log, validation: validation}
}

func (r *FiberRequest) AbortWithStatus(code int) {
	_ = r.instance.Status(code).Send(nil)
}

func (r *FiberRequest) AbortWithStatusJson(code int, jsonObj any) {
	_ = r.instance.Status(code).JSON(jsonObj)
}

func (r *FiberRequest) All() map[string]any {
	var (
		dataMap  = make(map[string]any)
		queryMap = make(map[string]string)
	)

	queryMap = r.instance.Queries()

	var mu sync.RWMutex
	for k, v := range queryMap {
		mu.Lock()
		dataMap[k] = v
		mu.Unlock()
	}
	for k, v := range r.postData {
		mu.Lock()
		dataMap[k] = v
		mu.Unlock()
	}

	return dataMap
}

func (r *FiberRequest) Bind(obj any) error {
	return r.instance.BodyParser(obj)
}

func (r *FiberRequest) Form(key string, defaultValue ...string) string {
	if len(defaultValue) == 0 {
		return r.instance.FormValue(key)
	}

	return r.instance.FormValue(key, defaultValue[0])
}

func (r *FiberRequest) File(name string) (filesystemcontract.File, error) {
	file, err := r.instance.FormFile(name)
	if err != nil {
		return nil, err
	}

	return filesystem.NewFileFromRequest(file)
}

func (r *FiberRequest) FullUrl() string {
	prefix := "https://"
	if r.instance.Secure() == false {
		prefix = "http://"
	}

	if r.instance.Hostname() == "" {
		return ""
	}

	return prefix + r.instance.Hostname() + r.instance.OriginalURL()
}

func (r *FiberRequest) Header(key string, defaultValue ...string) string {
	header := r.instance.Get(key)
	if header != "" {
		return header
	}

	if len(defaultValue) == 0 {
		return ""
	}

	return defaultValue[0]
}

func (r *FiberRequest) Headers() http.Header {
	// Fiber does not support http.Header
	//return r.instance.Request().Header
	return nil
}

func (r *FiberRequest) Host() string {
	return r.instance.Hostname()
}

func (r *FiberRequest) Json(key string, defaultValue ...string) string {
	data := make(map[string]any)
	err := sonic.Unmarshal(r.instance.Body(), &data)

	if err != nil {
		return ""
	}

	return cast.ToString(data[key])
}

func (r *FiberRequest) Method() string {
	return r.instance.Method()
}

func (r *FiberRequest) Next() {
	_ = r.instance.Next()
}

func (r *FiberRequest) Query(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return r.instance.Query(key, defaultValue[0])
	}

	return r.instance.Query(key)
}

func (r *FiberRequest) QueryInt(key string, defaultValue ...int) int {
	if r.instance.Query(key) != "" {
		return cast.ToInt(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *FiberRequest) QueryInt64(key string, defaultValue ...int64) int64 {
	if r.instance.Query(key) != "" {
		return cast.ToInt64(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *FiberRequest) QueryBool(key string, defaultValue ...bool) bool {
	if r.instance.Query(key) != "" {
		return stringToBool(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

func (r *FiberRequest) QueryArray(key string) []string {
	queries := r.instance.Queries()
	return []string{queries[key]}
}

func (r *FiberRequest) QueryMap(key string) map[string]string {
	return r.instance.Queries()
}

func (r *FiberRequest) Queries() map[string]string {
	return r.instance.Queries()
}

func (r *FiberRequest) Origin() *http.Request {
	// Fiber does not support http.Request
	return nil
}

func (r *FiberRequest) Path() string {
	return r.instance.Path()
}

func (r *FiberRequest) Input(key string, defaultValue ...string) string {
	if value, exist := r.postData[key]; exist {
		return cast.ToString(value)
	}

	if r.instance.Query(key) != "" {
		return r.instance.Query(key)
	}

	value := r.instance.Params(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return value
}

func (r *FiberRequest) InputInt(key string, defaultValue ...int) int {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt(value)
}

func (r *FiberRequest) InputInt64(key string, defaultValue ...int64) int64 {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt64(value)
}

func (r *FiberRequest) InputBool(key string, defaultValue ...bool) bool {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return stringToBool(value)
}

func (r *FiberRequest) Ip() string {
	return r.instance.IP()
}

func (r *FiberRequest) Route(key string) string {
	return r.instance.Params(key)
}

func (r *FiberRequest) RouteInt(key string) int {
	val := r.instance.Params(key)

	return cast.ToInt(val)
}

func (r *FiberRequest) RouteInt64(key string) int64 {
	val := r.instance.Params(key)

	return cast.ToInt64(val)
}

func (r *FiberRequest) Url() string {
	return r.instance.OriginalURL()
}

func (r *FiberRequest) Validate(rules map[string]string, options ...validatecontract.Option) (validatecontract.Validator, error) {
	if len(rules) == 0 {
		return nil, errors.New("rules can't be empty")
	}

	options = append(options, validation.Rules(rules), validation.CustomRules(r.validation.Rules()))
	generateOptions := validation.GenerateOptions(options)

	var v *validate.Validation
	dataFace, err := validate.FromRequest(r.Origin())
	if err != nil {
		return nil, err
	}
	if dataFace == nil {
		v = validate.NewValidation(dataFace)
	} else {
		if generateOptions["prepareForValidation"] != nil {
			if err := generateOptions["prepareForValidation"].(func(ctx httpcontract.Context, data validatecontract.Data) error)(r.ctx, validation.NewData(dataFace)); err != nil {
				return nil, err
			}
		}

		v = dataFace.Create()
	}

	validation.AppendOptions(v, generateOptions)

	return validation.NewValidator(v, dataFace), nil
}

func (r *FiberRequest) ValidateRequest(request httpcontract.FormRequest) (validatecontract.Errors, error) {
	if err := request.Authorize(r.ctx); err != nil {
		return nil, err
	}

	validator, err := r.Validate(request.Rules(r.ctx), validation.Messages(request.Messages(r.ctx)), validation.Attributes(request.Attributes(r.ctx)), func(options map[string]any) {
		options["prepareForValidation"] = request.PrepareForValidation
	})
	if err != nil {
		return nil, err
	}

	if err := validator.Bind(request); err != nil {
		return nil, err
	}

	return validator.Errors(), nil
}

func getPostData(ctx *FiberContext) (map[string]any, error) {
	if ctx.instance.Request().Body() == nil {
		return nil, nil
	}

	contentType := ctx.instance.Get("Content-Type")
	data := make(map[string]any)

	if contentType == "application/json" {
		bodyBytes := ctx.instance.Body()

		if err := sonic.Unmarshal(bodyBytes, &data); err != nil {
			return nil, fmt.Errorf("decode json [%v] error: %v", string(bodyBytes), err)
		}
	}
	if contentType == "multipart/form-data" {
		if form, err := ctx.instance.MultipartForm(); err == nil {
			for k, v := range form.Value {
				data[k] = strings.Join(v, ",")
			}
			for k, v := range form.File {
				if len(v) > 0 {
					data[k] = v[0]
				}
			}
		}
	}

	return data, nil
}

func stringToBool(value string) bool {
	return value == "1" || value == "true" || value == "on" || value == "yes"
}
