package fiber

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

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
	"github.com/goravel/framework/support/str"
	"github.com/goravel/framework/validation"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var contextRequestPool = sync.Pool{New: func() any {
	return &ContextRequest{
		log:        LogFacade,
		validation: ValidationFacade,
	}
}}

type ContextRequest struct {
	ctx        *Context
	instance   *fiber.Ctx
	httpBody   map[string]any
	log        log.Log
	validation contractsvalidate.Validation
}

func NewContextRequest(ctx *Context, log log.Log, validation contractsvalidate.Validation) contractshttp.ContextRequest {
	request := contextRequestPool.Get().(*ContextRequest)
	httpBody, err := getHttpBody(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("%+v", err))
	}
	request.ctx = ctx
	request.instance = ctx.instance
	request.httpBody = httpBody
	request.log = log
	request.validation = validation
	return request
}

func (r *ContextRequest) Abort(code ...int) {
	realCode := contractshttp.DefaultAbortStatus
	if len(code) > 0 {
		realCode = code[0]
	}

	if err := r.instance.SendStatus(realCode); err != nil {
		panic(err)
	}
}

// DEPRECATED: Use Abort instead
func (r *ContextRequest) AbortWithStatus(code int) {
	if err := r.instance.SendStatus(code); err != nil {
		panic(err)
	}
}

// DEPRECATED: Use Response().Json().Abort() instead
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

func (r *ContextRequest) BindQuery(obj any) error {
	return r.instance.QueryParser(obj)
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
	_, ok := r.ctx.Value(sessionKey).(contractsession.Session)
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

	if r.instance.Context().QueryArgs().Has(key) {
		return r.instance.Query(key)
	}

	return r.instance.Params(key, defaultValue...)
}

func (r *ContextRequest) InputArray(key string, defaultValue ...[]string) []string {
	if valueFromHttpBody := r.getValueFromHttpBody(key); valueFromHttpBody != nil {
		if value := cast.ToStringSlice(valueFromHttpBody); value == nil {
			return []string{}
		} else {
			return value
		}
	}

	if r.instance.Context().QueryArgs().Has(key) {
		valueSlice := r.instance.Context().QueryArgs().PeekMulti(key)
		value := make([]string, 0)
		for _, item := range valueSlice {
			if itemStr := utils.UnsafeString(item); itemStr != "" {
				value = append(value, itemStr)
			}
		}

		return value
	}

	if value, exist := r.instance.AllParams()[key]; exist {
		return str.Of(value).Split(",")
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	} else {
		return []string{}
	}
}

func (r *ContextRequest) InputMap(key string, defaultValue ...map[string]any) map[string]any {
	if valueFromHttpBody := r.getValueFromHttpBody(key); valueFromHttpBody != nil {
		return cast.ToStringMap(valueFromHttpBody)
	}

	if r.instance.Context().QueryArgs().Has(key) {
		valueStr := r.instance.Query(key)
		var value map[string]any
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			return map[string]any{}
		}

		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return map[string]any{}
}

func (r *ContextRequest) InputMapArray(key string, defaultValue ...[]map[string]any) []map[string]any {
	if valueFromHttpBody := r.getValueFromHttpBody(key); valueFromHttpBody != nil {
		var result = make([]map[string]any, 0)
		for _, item := range cast.ToSlice(valueFromHttpBody) {
			res, err := cast.ToStringMapE(item)
			if err != nil {
				return []map[string]any{}
			}
			result = append(result, res)
		}

		if len(result) == 0 {
			for _, item := range cast.ToStringSlice(valueFromHttpBody) {
				res, err := cast.ToStringMapE(item)
				if err != nil {
					return []map[string]any{}
				}
				result = append(result, res)
			}
		}

		return result
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return []map[string]any{}
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
	s, ok := r.ctx.Value(sessionKey).(contractsession.Session)
	if !ok {
		return nil
	}
	return s
}

func (r *ContextRequest) SetSession(session contractsession.Session) contractshttp.ContextRequest {
	r.ctx.WithValue(sessionKey, session)

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

	var options []contractsvalidate.Option
	if requestWithFilters, ok := request.(contractshttp.FormRequestWithFilters); ok {
		options = append(options, validation.Filters(requestWithFilters.Filters(r.ctx)))
	}
	if requestWithMessage, ok := request.(contractshttp.FormRequestWithMessages); ok {
		options = append(options, validation.Messages(requestWithMessage.Messages(r.ctx)))
	}
	if requestWithAttributes, ok := request.(contractshttp.FormRequestWithAttributes); ok {
		options = append(options, validation.Attributes(requestWithAttributes.Attributes(r.ctx)))
	}
	if prepareForValidation, ok := request.(contractshttp.FormRequestWithPrepareForValidation); ok {
		options = append(options, validation.PrepareForValidation(r.ctx, prepareForValidation.PrepareForValidation))
	}

	validator, err := r.Validate(request.Rules(r.ctx), options...)
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

		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &data); err != nil {
				return nil, fmt.Errorf("decode json [%v] error: %v", utils.UnsafeString(bodyBytes), err)
			}
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
