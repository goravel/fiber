package fiber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructToMap(t *testing.T) {
	data := struct {
		Name string
		Age  int
	}{
		Name: "test",
		Age:  18,
	}

	dataMap := structToMap(data)
	assert.Equal(t, "test", dataMap["Name"])
	assert.Equal(t, 18, dataMap["Age"])

	dataMap = structToMap(&data)
	assert.Equal(t, "test", dataMap["Name"])
	assert.Equal(t, 18, dataMap["Age"])
}

func TestFillShared(t *testing.T) {
	shared := map[string]any{
		"Name": "test",
	}
	data := map[string]any{
		"Age": 18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test", data["Name"])
	assert.Equal(t, 18, data["Age"])

	data = map[string]any{
		"Name": "test1",
		"Age":  18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test1", data["Name"])
	assert.Equal(t, 18, data["Age"])

	type Map map[string]any
	data = Map{
		"Age": 18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test", data["Name"])
	assert.Equal(t, 18, data["Age"])

	data = Map{
		"Name": "test1",
		"Age":  18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test1", data["Name"])
	assert.Equal(t, 18, data["Age"])
}
