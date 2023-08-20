package fiber

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gookit/validate"
	filesystemcontract "github.com/goravel/framework/contracts/filesystem"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/log"
	validatecontract "github.com/goravel/framework/contracts/validation"
	"github.com/goravel/framework/filesystem"
	"github.com/goravel/framework/validation"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type Request struct {
	ctx        *Context
	instance   *fiber.Ctx
	postData   map[string]any
	log        log.Log
	validation validatecontract.Validation
}

func NewRequest(ctx *Context, log log.Log, validation validatecontract.Validation) httpcontract.Request {
	postData, err := getPostData(ctx)
	if err != nil {
		LogFacade.Error(fmt.Sprintf("%+v", errors.Unwrap(err)))
	}

	return &Request{ctx: ctx, instance: ctx.instance, postData: postData, log: log, validation: validation}
}

func (r *Request) AbortWithStatus(code int) {
	_ = r.instance.SendStatus(code)
}

func (r *Request) AbortWithStatusJson(code int, jsonObj any) {
	_ = r.instance.Status(code).JSON(jsonObj)
}

func (r *Request) All() map[string]any {
	data := make(map[string]any)

	var mu sync.RWMutex
	for k, v := range r.instance.Queries() {
		mu.Lock()
		data[k] = v
		mu.Unlock()
	}
	for k, v := range r.postData {
		mu.Lock()
		data[k] = v
		mu.Unlock()
	}

	return data
}

func (r *Request) Bind(obj any) error {
	return r.instance.BodyParser(obj)
}

func (r *Request) Form(key string, defaultValue ...string) string {
	if len(defaultValue) == 0 {
		return r.instance.FormValue(key)
	}

	return r.instance.FormValue(key, defaultValue[0])
}

func (r *Request) File(name string) (filesystemcontract.File, error) {
	file, err := r.instance.FormFile(name)
	if err != nil {
		return nil, err
	}

	return filesystem.NewFileFromRequest(file)
}

func (r *Request) FullUrl() string {
	prefix := "https://"
	if !r.instance.Secure() {
		prefix = "http://"
	}

	if r.instance.Hostname() == "" {
		return ""
	}

	return prefix + r.instance.Hostname() + r.instance.OriginalURL()
}

func (r *Request) Header(key string, defaultValue ...string) string {
	header := r.instance.Get(key)
	if header != "" {
		return header
	}

	if len(defaultValue) == 0 {
		return ""
	}

	return defaultValue[0]
}

func (r *Request) Headers() http.Header {
	result := http.Header{}
	r.instance.Request().Header.VisitAll(func(key, value []byte) {
		result.Add(string(key), string(value))
	})

	return result
}

func (r *Request) Host() string {
	return r.instance.Hostname()
}

func (r *Request) Json(key string, defaultValue ...string) string {
	data := make(map[string]any)
	if err := sonic.Unmarshal(r.instance.Body(), &data); err != nil {
		if len(defaultValue) == 0 {
			return ""
		} else {
			return defaultValue[0]
		}
	}

	if value, exist := data[key]; exist {
		return cast.ToString(value)
	}

	if len(defaultValue) == 0 {
		return ""
	}

	return defaultValue[0]
}

func (r *Request) Method() string {
	return r.instance.Method()
}

func (r *Request) Next() {
	_ = r.instance.Next()
}

func (r *Request) Query(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return r.instance.Query(key, defaultValue[0])
	}

	return r.instance.Query(key)
}

func (r *Request) QueryInt(key string, defaultValue ...int) int {
	if r.instance.Query(key) != "" {
		return cast.ToInt(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *Request) QueryInt64(key string, defaultValue ...int64) int64 {
	if r.instance.Query(key) != "" {
		return cast.ToInt64(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *Request) QueryBool(key string, defaultValue ...bool) bool {
	if r.instance.Query(key) != "" {
		return stringToBool(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

func (r *Request) QueryArray(key string) []string {
	var queries []string
	r.instance.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		if key == string(k) {
			queries = append(queries, string(v))
		}
	})

	return queries
}

func (r *Request) QueryMap(key string) map[string]string {
	queries := make(map[string]string)
	r.instance.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		matches := regexp.MustCompile(`^` + key + `\[(.+)\]$`).FindSubmatch(k)
		if len(matches) > 0 {
			queries[string(matches[1])] = string(v)
		}
	})

	return queries
}

func (r *Request) Queries() map[string]string {
	return r.instance.Queries()
}

func (r *Request) Origin() *http.Request {
	var req http.Request
	_ = fasthttpadaptor.ConvertRequest(r.instance.Context(), &req, true)

	return &req
}

func (r *Request) Path() string {
	return r.instance.Path()
}

func (r *Request) Input(key string, defaultValue ...string) string {
	keys := strings.Split(key, ".")
	current := r.postData
	for _, k := range keys {
		value, found := current[k]
		if found {
			if nestedMap, isMap := value.(map[string]any); isMap {
				current = nestedMap
			} else {
				return cast.ToString(value)
			}
		}
	}

	if r.instance.Query(key) != "" {
		return r.instance.Query(key)
	}

	value := r.instance.Params(key)
	if len(value) == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return value
}

func (r *Request) InputArray(key string, defaultValue ...[]string) []string {
	keys := strings.Split(key, ".")
	current := r.postData
	for _, k := range keys {
		value, found := current[k]
		if !found {
			return []string{}
		}
		if nestedMap, isMap := value.(map[string]any); isMap {
			current = nestedMap
		} else {
			return cast.ToStringSlice(value)
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return []string{}
	}
}

func (r *Request) InputMap(key string, defaultValue ...map[string]string) map[string]string {
	keys := strings.Split(key, ".")
	current := r.postData
	for _, k := range keys {
		value, found := current[k]
		if !found {
			return map[string]string{}
		}
		if nestedMap, isMap := value.(map[string]string); isMap {
			current = cast.ToStringMap(nestedMap)
		} else {
			return cast.ToStringMapString(value)
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return map[string]string{}
	}
}

func (r *Request) InputInt(key string, defaultValue ...int) int {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt(value)
}

func (r *Request) InputInt64(key string, defaultValue ...int64) int64 {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt64(value)
}

func (r *Request) InputBool(key string, defaultValue ...bool) bool {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return stringToBool(value)
}

func (r *Request) Ip() string {
	return r.instance.IP()
}

func (r *Request) Route(key string) string {
	return r.instance.Params(key)
}

func (r *Request) RouteInt(key string) int {
	val := r.instance.Params(key)

	return cast.ToInt(val)
}

func (r *Request) RouteInt64(key string) int64 {
	val := r.instance.Params(key)

	return cast.ToInt64(val)
}

func (r *Request) Url() string {
	return r.instance.OriginalURL()
}

func (r *Request) Validate(rules map[string]string, options ...validatecontract.Option) (validatecontract.Validator, error) {
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

func (r *Request) ValidateRequest(request httpcontract.FormRequest) (validatecontract.Errors, error) {
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

func getPostData(ctx *Context) (map[string]any, error) {
	if len(ctx.instance.Request().Body()) == 0 {
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

	if contentType == "application/x-www-form-urlencoded" {
		args := ctx.instance.Request().PostArgs()
		args.VisitAll(func(key, value []byte) {
			data[string(key)] = string(value)
		})
	}

	return data, nil
}

func stringToBool(value string) bool {
	return value == "1" || value == "true" || value == "on" || value == "yes"
}
