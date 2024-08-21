package fiber

import (
	"io"
	"net/http"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	configmocks "github.com/goravel/framework/mocks/config"
	httpmocks "github.com/goravel/framework/mocks/http"
	"github.com/goravel/framework/support/file"
	"github.com/stretchr/testify/assert"
)

func TestView_Make(t *testing.T) {
	var (
		err        error
		fiber      *Route
		req        *http.Request
		mockConfig *configmocks.Config
		mockView   *httpmocks.View
	)

	assert.Nil(t, file.Create("resources/views/empty.tmpl", `{{ define "empty.tmpl" }}
1
{{ end }}
`))
	assert.Nil(t, file.Create("resources/views/data.tmpl", `{{ define "data.tmpl" }}
{{ .Name }}
{{ .Age }}
{{ end }}
`))

	beforeEach := func() {
		mockConfig = &configmocks.Config{}
		mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		ConfigFacade = mockConfig

		mockView = &httpmocks.View{}
		ViewFacade = mockView
	}
	tests := []struct {
		name        string
		method      string
		url         string
		setup       func(method, url string) error
		expectCode  int
		expectBody  string
		expectPanic bool
	}{
		{
			name:   "data is empty, shared is empty",
			method: "GET",
			url:    "/make/empty",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(nil).Once()

				fiber.Get("/make/empty", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("empty.tmpl")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\n1\n",
		},
		{
			name:   "data is empty, shared is not empty",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
					"Age":  18,
				}).Once()

				fiber.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "data is not empty, shared is not empty",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				fiber.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", map[string]any{
						"Age": 18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "data is not empty, shared is not empty, and data contains shared key",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				fiber.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", map[string]any{
						"Name": "test1",
						"Age":  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest1\n18\n",
		},
		{
			name:   "data is struct, shared is not empty, and data contains shared key",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				fiber.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", struct {
						Name string
						Age  int
					}{
						Name: "test1",
						Age:  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest1\n18\n",
		},
		{
			name:   "data is []string",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(nil).Once()

				fiber.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					assert.Panics(t, func() {
						ctx.Response().View().Make("data.tmpl", []string{"test"})
					})

					return nil
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
			mockView.AssertExpectations(t)
		})
	}

	assert.Nil(t, file.Remove("resources"))
}

func TestView_First(t *testing.T) {
	var (
		err        error
		fiber      *Route
		req        *http.Request
		mockConfig *configmocks.Config
		mockView   *httpmocks.View
	)

	assert.Nil(t, file.Create("resources/views/empty.tmpl", `{{ define "empty.tmpl" }}
1
{{ end }}
`))
	assert.Nil(t, file.Create("resources/views/data.tmpl", `{{ define "data.tmpl" }}
{{ .Name }}
{{ .Age }}
{{ end }}
`))

	beforeEach := func() {
		mockConfig = &configmocks.Config{}
		mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		ConfigFacade = mockConfig

		mockView = &httpmocks.View{}
		ViewFacade = mockView
	}
	tests := []struct {
		name        string
		method      string
		url         string
		setup       func(method, url string) error
		expectCode  int
		expectBody  string
		expectPanic bool
	}{
		{
			name:   "found the first view",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(true).Once()
				mockView.On("GetShared").Return(nil).Once()

				fiber.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\n1\n",
		},
		{
			name:   "found the second view",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(false).Once()
				mockView.On("Exists", "data.tmpl").Return(true).Once()
				mockView.On("GetShared").Return(nil).Once()

				fiber.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"}, map[string]any{
						"Name": "test",
						"Age":  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "no view found",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(false).Once()
				mockView.On("Exists", "data.tmpl").Return(false).Once()

				fiber.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					assert.Panics(t, func() {
						ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"}, map[string]any{
							"Name": "test",
							"Age":  18,
						})
					})

					return nil
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
			mockView.AssertExpectations(t)
		})
	}

	assert.Nil(t, file.Remove("resources"))
}

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
