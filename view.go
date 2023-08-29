package fiber

import (
	"fmt"
	"reflect"

	"github.com/gofiber/fiber/v2"
)

type View struct {
	instance *fiber.Ctx
}

func NewView(instance *fiber.Ctx) *View {
	return &View{instance: instance}
}

func (receive *View) Make(view string, data ...any) {
	shared := ViewFacade.GetShared()
	if len(data) == 0 {
		err := receive.instance.Render("index", shared)
		if err != nil {
			panic(err)
		}
	} else {
		dataType := reflect.TypeOf(data[0])
		switch dataType.Kind() {
		case reflect.Struct:
			dataMap := structToMap(data[0])
			for key, value := range dataMap {
				shared[key] = value
			}
			err := receive.instance.Render(view, shared)
			if err != nil {
				panic(err)
			}
		case reflect.Map:
			fillShared(data[0], shared)
			err := receive.instance.Render(view, data[0])
			if err != nil {
				panic(err)
			}
		default:
			panic(fmt.Sprintf("make %s view failed, data must be map or struct", view))
		}
	}
}

func (receive *View) First(views []string, data ...any) {
	for _, view := range views {
		if ViewFacade.Exists(view) {
			receive.Make(view, data...)
			return
		}
	}
}

func structToMap(data any) map[string]any {
	res := make(map[string]any)
	modelType := reflect.TypeOf(data)
	modelValue := reflect.ValueOf(data)

	if modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
		modelValue = modelValue.Elem()
	}

	for i := 0; i < modelType.NumField(); i++ {
		dbColumn := modelType.Field(i).Name
		if modelValue.Field(i).Kind() == reflect.Pointer {
			if modelValue.Field(i).IsNil() {
				res[dbColumn] = nil
			} else {
				res[dbColumn] = modelValue.Field(i).Elem().Interface()
			}
		} else {
			res[dbColumn] = modelValue.Field(i).Interface()
		}
	}

	return res
}

func fillShared(data any, shared map[string]any) {
	dataValue := reflect.ValueOf(data)
	keys := dataValue.MapKeys()
	for key, value := range shared {
		exist := false
		for _, k := range keys {
			if k.String() == key {
				exist = true
				break
			}
		}
		if !exist {
			dataValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}
}
