package fiber

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gookit/validate"
	contractsfilesystem "github.com/goravel/framework/contracts/filesystem"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/log"
	contractsession "github.com/goravel/framework/contracts/session"
	contractsvalidate "github.com/goravel/framework/contracts/validation"
	"github.com/goravel/framework/filesystem"
	"github.com/goravel/framework/support/json"
	"github.com/goravel/framework/validation"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type ContextRequest struct {
	ctx        *Context
	instance   *fiber.Ctx
	httpBody   map[string]any
	log        log.Log
	validation contractsvalidate.Validation
}

func NewContextRequest(ctx *Context, log log.Log, validation contractsvalidate.Validation) contractshttp.ContextRequest {
	httpBody, err := getHttpBody(ctx)
	if err != nil {
		LogFacade.Error(fmt.Sprintf("%+v", errors.Unwrap(err)))
	}

	return &ContextRequest{ctx: ctx, instance: ctx.instance, httpBody: httpBody, log: log, validation: validation}
}

func (r *ContextRequest) AbortWithStatus(code int) {
	if err := r.instance.SendStatus(code); err != nil {
		panic(err)
	}
}

func (r *ContextRequest) AbortWithStatusJson(code int, jsonObj any) {
	if err := r.instance.Status(code).JSON(jsonObj); err != nil {
		panic(err)
	}
}

func (r *ContextRequest) All() map[string]any {
	data := make(map[string]any)

	for k, v := range r.instance.AllParams() {
		data[k] = v
	}
	for k, v := range r.instance.Queries() {
		data[k] = v
	}
	for k, v := range r.httpBody {
		data[k] = v
	}

	return data
}

func (r *ContextRequest) Bind(obj any) error {
	return r.instance.BodyParser(obj)
}

func (r *ContextRequest) Cookie(key string, defaultValue ...string) string {
	return r.instance.Cookies(key, defaultValue...)
}

func (r *ContextRequest) Form(key string, defaultValue ...string) string {
	if len(defaultValue) == 0 {
		return r.instance.FormValue(key)
	}

	return r.instance.FormValue(key, defaultValue[0])
}

func (r *ContextRequest) File(name string) (contractsfilesystem.File, error) {
	file, err := r.instance.FormFile(name)
	if err != nil {
		return nil, err
	}

	return filesystem.NewFileFromRequest(file)
}

func (r *ContextRequest) FullUrl() string {
	prefix := "https://"
	if !r.instance.Secure() {
		prefix = "http://"
	}

	if r.instance.Hostname() == "" {
		return ""
	}

	return prefix + r.instance.Hostname() + r.instance.OriginalURL()
}

func (r *ContextRequest) Header(key string, defaultValue ...string) string {
	header := r.instance.Get(key)
	if header != "" {
		return header
	}

	if len(defaultValue) == 0 {
		return ""
	}

	return defaultValue[0]
}

func (r *ContextRequest) Headers() http.Header {
	result := http.Header{}
	r.instance.Request().Header.VisitAll(func(key, value []byte) {
		result.Add(utils.UnsafeString(key), utils.UnsafeString(value))
	})

	return result
}

func (r *ContextRequest) Host() string {
	return r.instance.Hostname()
}

func (r *ContextRequest) HasSession() bool {
	_, ok := r.ctx.Value("session").(contractsession.Session)
	return ok
}

func (r *ContextRequest) Json(key string, defaultValue ...string) string {
	data := make(map[string]any)
	if err := json.Unmarshal(r.instance.Body(), &data); err != nil {
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

func (r *ContextRequest) Method() string {
	return r.instance.Method()
}

func (r *ContextRequest) Next() {
	if err := r.instance.Next(); err != nil {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			if err := r.instance.Status(fiberErr.Code).SendString(fiberErr.Message); err == nil {
				return
			}
		}

		panic(err)
	}
}

func (r *ContextRequest) Query(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return r.instance.Query(key, defaultValue[0])
	}

	return r.instance.Query(key)
}

func (r *ContextRequest) QueryInt(key string, defaultValue ...int) int {
	if r.instance.Query(key) != "" {
		return cast.ToInt(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *ContextRequest) QueryInt64(key string, defaultValue ...int64) int64 {
	if r.instance.Query(key) != "" {
		return cast.ToInt64(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

func (r *ContextRequest) QueryBool(key string, defaultValue ...bool) bool {
	if r.instance.Query(key) != "" {
		return stringToBool(r.instance.Query(key))
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

func (r *ContextRequest) QueryArray(key string) []string {
	var queries []string
	r.instance.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		if key == utils.UnsafeString(k) {
			queries = append(queries, utils.UnsafeString(v))
		}
	})

	return queries
}

func (r *ContextRequest) QueryMap(key string) map[string]string {
	queries := make(map[string]string)
	r.instance.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		matches := regexp.MustCompile(`^` + key + `\[(.+)\]$`).FindSubmatch(k)
		if len(matches) > 0 {
			queries[utils.UnsafeString(matches[1])] = utils.UnsafeString(v)
		}
	})

	return queries
}

func (r *ContextRequest) Queries() map[string]string {
	return r.instance.Queries()
}

func (r *ContextRequest) Origin() *http.Request {
	var req http.Request
	if err := fasthttpadaptor.ConvertRequest(r.instance.Context(), &req, true); err != nil {
		panic(err)
	}

	return &req
}

func (r *ContextRequest) Path() string {
	return r.instance.Path()
}

func (r *ContextRequest) Input(key string, defaultValue ...string) string {
	valueFromHttpBody := r.getValueFromHttpBody(key)
	if valueFromHttpBody != nil {
		switch reflect.ValueOf(valueFromHttpBody).Kind() {
		case reflect.Map:
			valueFromHttpBodyByte, err := json.Marshal(valueFromHttpBody)
			if err != nil {
				return ""
			}

			return utils.UnsafeString(valueFromHttpBodyByte)
		case reflect.Slice:
			return strings.Join(cast.ToStringSlice(valueFromHttpBody), ",")
		default:
			return cast.ToString(valueFromHttpBody)
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

func (r *ContextRequest) InputArray(key string, defaultValue ...[]string) []string {
	if valueFromHttpBody := r.getValueFromHttpBody(key); valueFromHttpBody != nil {
		return cast.ToStringSlice(valueFromHttpBody)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return []string{}
	}
}

func (r *ContextRequest) InputMap(key string, defaultValue ...map[string]string) map[string]string {
	if valueFromHttpBody := r.getValueFromHttpBody(key); valueFromHttpBody != nil {
		return cast.ToStringMapString(valueFromHttpBody)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return map[string]string{}
	}
}

func (r *ContextRequest) InputInt(key string, defaultValue ...int) int {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt(value)
}

func (r *ContextRequest) InputInt64(key string, defaultValue ...int64) int64 {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return cast.ToInt64(value)
}

func (r *ContextRequest) InputBool(key string, defaultValue ...bool) bool {
	value := r.Input(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return stringToBool(value)
}

func (r *ContextRequest) Ip() string {
	return r.instance.IP()
}

func (r *ContextRequest) Route(key string) string {
	return r.instance.Params(key)
}

func (r *ContextRequest) RouteInt(key string) int {
	val := r.instance.Params(key)

	return cast.ToInt(val)
}

func (r *ContextRequest) RouteInt64(key string) int64 {
	val := r.instance.Params(key)

	return cast.ToInt64(val)
}

func (r *ContextRequest) Session() contractsession.Session {
	s, ok := r.ctx.Value("session").(contractsession.Session)
	if !ok {
		return nil
	}
	return s
}

func (r *ContextRequest) SetSession(session contractsession.Session) contractshttp.ContextRequest {
	r.ctx.WithValue("session", session)

	return r
}

func (r *ContextRequest) Url() string {
	return r.instance.OriginalURL()
}

func (r *ContextRequest) Validate(rules map[string]string, options ...contractsvalidate.Option) (contractsvalidate.Validator, error) {
	if len(rules) == 0 {
		return nil, errors.New("rules can't be empty")
	}

	options = append(options, validation.Rules(rules), validation.CustomRules(r.validation.Rules()))

	dataFace, err := validate.FromRequest(r.ctx.Request().Origin())
	if err != nil {
		return nil, err
	}

	for key, query := range r.instance.Queries() {
		if _, exist := dataFace.Get(key); !exist {
			if _, err := dataFace.Set(key, query); err != nil {
				return nil, err
			}
		}
	}

	for key, param := range r.instance.AllParams() {
		if _, exist := dataFace.Get(key); !exist {
			if _, err := dataFace.Set(key, param); err != nil {
				return nil, err
			}
		}
	}

	return r.validation.Make(dataFace, rules, options...)
}

func (r *ContextRequest) ValidateRequest(request contractshttp.FormRequest) (contractsvalidate.Errors, error) {
	if err := request.Authorize(r.ctx); err != nil {
		return nil, err
	}

	filters := make(map[string]string)
	val := reflect.Indirect(reflect.ValueOf(request))
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Type().Field(i)
		form := field.Tag.Get("form")
		filter := field.Tag.Get("filter")
		if len(form) > 0 && len(filter) > 0 {
			filters[form] = filter
		}
	}

	validator, err := r.Validate(request.Rules(r.ctx), validation.Messages(request.Messages(r.ctx)), validation.Attributes(request.Attributes(r.ctx)), func(options map[string]any) {
		options["prepareForValidation"] = request.PrepareForValidation
		options["filters"] = filters
	})
	if err != nil {
		return nil, err
	}

	if err := validator.Bind(request); err != nil {
		return nil, err
	}

	return validator.Errors(), nil
}

func (r *ContextRequest) getValueFromHttpBody(key string) any {
	if r.httpBody == nil {
		return nil
	}

	var current any
	current = r.httpBody
	keys := strings.Split(key, ".")
	for _, k := range keys {
		currentValue := reflect.ValueOf(current)
		switch currentValue.Kind() {
		case reflect.Map:
			if value := currentValue.MapIndex(reflect.ValueOf(k)); value.IsValid() {
				current = value.Interface()
			} else {
				if value := currentValue.MapIndex(reflect.ValueOf(k + "[]")); value.IsValid() {
					current = value.Interface()
				} else {
					return nil
				}
			}
		case reflect.Slice:
			if number, err := strconv.Atoi(k); err == nil {
				return cast.ToStringSlice(current)[number]
			} else {
				return nil
			}
		}
	}

	return current
}

func getHttpBody(ctx *Context) (map[string]any, error) {
	if len(ctx.instance.Request().Body()) == 0 {
		return nil, nil
	}

	contentType := ctx.instance.Get("Content-Type")
	data := make(map[string]any)

	if strings.Contains(contentType, "application/json") {
		bodyBytes := ctx.instance.Body()

		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			return nil, fmt.Errorf("decode json [%v] error: %v", utils.UnsafeString(bodyBytes), err)
		}
	}

	if strings.Contains(contentType, "multipart/form-data") {
		if form, err := ctx.instance.MultipartForm(); err == nil {
			for k, v := range form.Value {
				if len(v) > 1 {
					data[k] = v
				} else if len(v) == 1 {
					data[k] = v[0]
				}
			}
			for k, v := range form.File {
				if len(v) > 1 {
					data[k] = v
				} else if len(v) == 1 {
					data[k] = v[0]
				}
			}
		}
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		args := ctx.instance.Request().PostArgs()
		args.VisitAll(func(key, value []byte) {
			if existValue, exist := data[utils.UnsafeString(key)]; exist {
				data[utils.UnsafeString(key)] = append([]string{cast.ToString(existValue)}, utils.UnsafeString(value))
			} else {
				data[utils.UnsafeString(key)] = utils.UnsafeString(value)
			}
		})
	}

	return data, nil
}

func stringToBool(value string) bool {
	return value == "1" || value == "true" || value == "on" || value == "yes"
}
