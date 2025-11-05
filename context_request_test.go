package fiber

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	frameworkfilesystem "github.com/goravel/framework/filesystem"
	foundationjson "github.com/goravel/framework/foundation/json"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mocksfilesystem "github.com/goravel/framework/mocks/filesystem"
	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/goravel/framework/session"
	"github.com/goravel/framework/support/json"
	"github.com/goravel/framework/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ContextRequestSuite struct {
	suite.Suite
	route      *Route
	mockConfig *mocksconfig.Config
}

func TestContextRequestSuite(t *testing.T) {
	suite.Run(t, new(ContextRequestSuite))
}

func (s *ContextRequestSuite) SetupTest() {
	s.mockConfig = mocksconfig.NewConfig(s.T())
	s.mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Once()
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	s.mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	s.mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	s.mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	s.mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
	ValidationFacade = validation.NewValidation()

	s.route = &Route{
		config: s.mockConfig,
		driver: "fiber",
	}
	err := s.route.init(nil)
	s.Require().Nil(err)
}

func (s *ContextRequestSuite) TestAll() {
	s.route.Get("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	req, err := http.NewRequest("GET", "/all", nil)
	s.Require().Nil(err)
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_GetWithQuery() {
	s.route.Get("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	req, err := http.NewRequest("GET", "/all?a=1&a=2&b=3", nil)
	s.Require().Nil(err)
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"b\":\"3\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithQueryAndForm() {
	s.route.Post("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	err := writer.WriteField("b", "4")
	s.Require().NoError(err)

	err = writer.WriteField("e", "e")
	s.Require().NoError(err)

	err = writer.Close()
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", "/all?a=1&a=2&b=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"b\":\"4\",\"e\":\"e\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithQuery() {
	s.route.Post("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	req, err := http.NewRequest("POST", "/all?a=1&a=2&b=3", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "multipart/form-data;boundary=0")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"b\":\"3\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithJson() {
	s.route.Post("/all", func(ctx contractshttp.Context) contractshttp.Response {
		all := ctx.Request().All()
		type Test struct {
			Name string
			Age  int
		}
		var test Test
		_ = ctx.Request().Bind(&test)

		return ctx.Response().Success().Json(contractshttp.Json{
			"all":  all,
			"name": test.Name,
			"age":  test.Age,
		})
	})

	payload := strings.NewReader(`{
		"Name": "goravel",
		"Age": 1
	}`)
	req, err := http.NewRequest("POST", "/all?a=1&a=2&name=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"age\":1,\"all\":{\"Age\":1,\"Name\":\"goravel\",\"a\":\"1,2\",\"name\":\"3\"},\"name\":\"goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithErrorJson() {
	mockLog := &mockslog.Log{}
	LogFacade = mockLog
	mockLog.EXPECT().Error(mock.Anything).Twice()

	s.route.Post("/all", func(ctx contractshttp.Context) contractshttp.Response {
		all := ctx.Request().All()
		type Test struct {
			Name string
			Age  int
		}
		var test Test
		_ = ctx.Request().Bind(&test)

		return ctx.Response().Success().Json(contractshttp.Json{
			"all":  all,
			"name": test.Name,
			"age":  test.Age,
		})
	})

	payload := strings.NewReader(`{
		"Name": "goravel",
		"Age": 1,
	}`)
	req, err := http.NewRequest("POST", "/all?a=1&a=2&name=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"age\":0,\"all\":{\"a\":\"1,2\",\"name\":\"3\"},\"name\":\"\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithEmptyJson() {
	s.route.Post("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	req, err := http.NewRequest("POST", "/all?a=1&a=2&name=3", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"name\":\"3\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PostWithMiddleware() {
	s.route.Middleware(testAllMiddleware()).Post("/all-with-middleware", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all":        ctx.Request().All(),
			"middleware": ctx.Value("all"),
		})
	})

	payload := strings.NewReader(`{
		"Name": "goravel",
		"Age": 1
	}`)
	req, err := http.NewRequest("POST", "/all-with-middleware?a=1&a=2&name=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"Age\":1,\"Name\":\"goravel\",\"a\":\"1,2\",\"name\":\"3\"},\"middleware\":{\"Age\":1,\"Name\":\"goravel\",\"a\":\"1,2\",\"name\":\"3\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_PutWithJson() {
	s.route.Put("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	payload := strings.NewReader(`{
		"b": 4,
		"e": "e"
	}`)
	req, err := http.NewRequest("PUT", "/all?a=1&a=2&b=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"b\":4,\"e\":\"e\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestAll_DeleteWithJson() {
	s.route.Delete("/all", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	payload := strings.NewReader(`{
		"b": 4,
		"e": "e"
	}`)
	req, err := http.NewRequest("DELETE", "/all?a=1&a=2&b=3", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1,2\",\"b\":4,\"e\":\"e\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestBind_Json() {
	s.route.Post("/bind/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		id := ctx.Request().Input("id")
		var data struct {
			Name string `form:"name" json:"name"`
		}
		_ = ctx.Request().Bind(&data)
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":   id,
			"name": data.Name,
		})
	})

	payload := strings.NewReader(`{
		"name": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/bind/json/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"2\",\"name\":\"Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestBind_Form() {
	s.route.Post("/bind/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		id := ctx.Request().Input("id")
		var data struct {
			Name string `form:"name" json:"name"`
		}
		_ = ctx.Request().Bind(&data)
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":   id,
			"name": data.Name,
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.WriteField("name", "Goravel")
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/bind/form/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"2\",\"name\":\"Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestBind_Query() {
	s.route.Post("/bind/query/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	req, err := http.NewRequest("POST", "/bind/query/1?id=2", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"2\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestBindQueryToStruct() {
	s.route.Get("/bind/query/struct", func(ctx contractshttp.Context) contractshttp.Response {
		type Test struct {
			ID string `form:"id"`
		}
		var test Test
		_ = ctx.Request().BindQuery(&test)
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": test.ID,
		})
	})

	req, err := http.NewRequest("GET", "/bind/query/struct?id=2", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"2\"}", body)
	s.Equal(http.StatusOK, code)

	// complex struct
	s.route.Get("/bind/query/struct/complex", func(ctx contractshttp.Context) contractshttp.Response {
		type Person struct {
			Name string `form:"name" json:"name"`
			Age  int    `form:"age" json:"age"`
		}
		type Test struct {
			ID      string   `form:"id"`
			Persons []Person `form:"persons"`
		}
		var test Test
		_ = ctx.Request().BindQuery(&test)
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":      test.ID,
			"persons": test.Persons,
		})
	})

	req, err = http.NewRequest("GET", "/bind/query/struct/complex?id=2&persons[0][name]=John&persons[0][age]=30&persons[1][name]=Doe&persons[1][age]=40", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ = s.request(req)

	s.Equal("{\"id\":\"2\",\"persons\":[{\"name\":\"John\",\"age\":30},{\"name\":\"Doe\",\"age\":40}]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestBind_ThenInput() {
	s.route.Post("/bind/input", func(ctx contractshttp.Context) contractshttp.Response {
		type Test struct {
			Name string
		}
		var test Test
		_ = ctx.Request().Bind(&test)
		return ctx.Response().Success().Json(contractshttp.Json{
			"name":  test.Name,
			"name1": ctx.Request().Input("Name"),
		})
	})

	payload := strings.NewReader(`{
		"Name": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/bind/input", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel\",\"name1\":\"Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestCookie() {
	s.route.Get("/cookie", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"goravel": ctx.Request().Cookie("goravel"),
		})
	})

	payload := strings.NewReader(`{
		"name": "Goravel"
	}`)
	req, err := http.NewRequest("GET", "/cookie", payload)
	s.Require().Nil(err)

	req.AddCookie(&http.Cookie{
		Name:  "goravel",
		Value: "goravel",
	})
	code, body, _, _ := s.request(req)

	s.Equal("{\"goravel\":\"goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestCookie_Default() {
	s.route.Get("/cookie/default", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"goravel": ctx.Request().Cookie("goravel", "default value"),
		})
	})

	req, err := http.NewRequest("GET", "/cookie/default", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"goravel\":\"default value\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestFile() {
	s.route.Post("/file", func(ctx contractshttp.Context) contractshttp.Response {
		s.mockConfig.On("GetString", "app.name").Return("goravel").Once()
		s.mockConfig.On("GetString", "filesystems.default").Return("local").Once()
		frameworkfilesystem.ConfigFacade = s.mockConfig

		mockStorage := &mocksfilesystem.Storage{}
		mockDriver := &mocksfilesystem.Driver{}
		mockStorage.On("Disk", "local").Return(mockDriver).Once()
		frameworkfilesystem.StorageFacade = mockStorage

		fileInfo, err := ctx.Request().File("file")

		mockDriver.On("PutFile", "test", fileInfo).Return("test/README.md", nil).Once()
		mockStorage.On("Exists", "test/README.md").Return(true).Once()

		if err != nil {
			return ctx.Response().Success().String("get file error")
		}
		filePath, err := fileInfo.Store("test")
		if err != nil {
			return ctx.Response().Success().String("store file error: " + err.Error())
		}

		extension, err := fileInfo.Extension()
		if err != nil {
			return ctx.Response().Success().String("get file extension error: " + err.Error())
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"exist":              mockStorage.Exists(filePath),
			"hash_name_length":   len(fileInfo.HashName()),
			"hash_name_length1":  len(fileInfo.HashName("test")),
			"file_path_length":   len(filePath),
			"extension":          extension,
			"original_name":      fileInfo.GetClientOriginalName(),
			"original_extension": fileInfo.GetClientOriginalExtension(),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	readme, err := os.Open("./README.md")
	s.Require().Nil(err)
	defer func() {
		_ = readme.Close()
	}()

	part1, err := writer.CreateFormFile("file", filepath.Base("./README.md"))
	s.Require().Nil(err)

	_, err = io.Copy(part1, readme)
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/file", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"exist\":true,\"extension\":\"txt\",\"file_path_length\":14,\"hash_name_length\":44,\"hash_name_length1\":49,\"original_extension\":\"md\",\"original_name\":\"README.md\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestFiles() {
	s.route.Post("/files", func(ctx contractshttp.Context) contractshttp.Response {
		s.mockConfig.On("GetString", "app.name").Return("goravel").Twice()
		s.mockConfig.On("GetString", "filesystems.default").Return("local").Twice()
		frameworkfilesystem.ConfigFacade = s.mockConfig

		mockStorage := &mocksfilesystem.Storage{}
		mockDriver := &mocksfilesystem.Driver{}
		mockStorage.On("Disk", "local").Return(mockDriver).Twice()
		frameworkfilesystem.StorageFacade = mockStorage

		filesInfo, err := ctx.Request().Files("files")
		if err != nil {
			return ctx.Response().Success().String("get files error")
		}

		response := contractshttp.Json{
			"file_count": len(filesInfo),
		}
		for _, fileInfo := range filesInfo {

			fp := filepath.Join("test", fileInfo.GetClientOriginalName())
			mockDriver.On("PutFile", "test", fileInfo).Return(fp, nil).Once()
			mockStorage.On("Exists", fp).Return(true).Once()

			filePath, err := fileInfo.Store("test")
			if err != nil {
				return ctx.Response().Success().String("store file error: " + err.Error())
			}

			extension, err := fileInfo.Extension()
			if err != nil {
				return ctx.Response().Success().String("get file extension error: " + err.Error())
			}

			response[fileInfo.GetClientOriginalName()] = contractshttp.Json{
				"exist":             mockStorage.Exists(filePath),
				"hash_name_length":  len(fileInfo.HashName()),
				"hash_name_length1": len(fileInfo.HashName("test")),
				"file_path_length":  len(filePath),
				"extension":         extension,
			}
		}
		return ctx.Response().Success().Json(response)
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	readme, err := os.Open("./README.md")
	s.Require().Nil(err)
	defer func() {
		_ = readme.Close()
	}()

	route, err := os.Open("./route.go")
	s.Require().Nil(err)
	defer func() {
		_ = route.Close()
	}()

	part1, err := writer.CreateFormFile("files", filepath.Base("./README.md"))
	s.Require().Nil(err)
	_, err = io.Copy(part1, readme)
	s.Require().Nil(err)

	part2, err := writer.CreateFormFile("files", filepath.Base("./route.go"))
	s.Require().Nil(err)
	_, err = io.Copy(part2, route)
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/files", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.JSONEq(`{
     "file_count": 2,
	"README.md": {
        "exist": true,
        "extension": "txt",
        "file_path_length": 14,
        "hash_name_length": 44,
        "hash_name_length1": 49
    },
    "route.go": {
        "exist": true,
        "extension": "txt",
        "file_path_length": 13,
        "hash_name_length": 44,
        "hash_name_length1": 49
    }
}`, body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestHeaders() {
	s.route.Get("/headers", func(ctx contractshttp.Context) contractshttp.Response {
		str, _ := json.Marshal(ctx.Request().Headers())
		return ctx.Response().Success().String(string(str))
	})

	req, err := http.NewRequest("GET", "/headers", nil)
	s.Require().Nil(err)

	req.Header.Set("Hello", "Goravel")
	code, body, _, _ := s.request(req)

	s.Equal("{\"Content-Length\":[\"0\"],\"Hello\":[\"Goravel\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestMethods() {
	s.route.Get("/methods/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":          ctx.Request().Input("id"),
			"name":        ctx.Request().Query("name", "Hello"),
			"header":      ctx.Request().Header("Hello", "World"),
			"method":      ctx.Request().Method(),
			"origin_path": ctx.Request().OriginPath(),
			"path":        ctx.Request().Path(),
			"url":         ctx.Request().Url(),
			"full_url":    ctx.Request().FullUrl(),
			"ip":          ctx.Request().Ip(),
		})
	})

	req, err := http.NewRequest("GET", "/methods/1?name=Goravel", nil)
	s.Require().Nil(err)

	req.Header.Set("Hello", "Goravel")
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	code, body, _, _ := s.request(req)

	s.Equal("{\"full_url\":\"\",\"header\":\"Goravel\",\"id\":\"1\",\"ip\":\"1.1.1.1\",\"method\":\"GET\",\"name\":\"Goravel\",\"origin_path\":\"/methods/{id}\",\"path\":\"/methods/1\",\"url\":\"/methods/1?name=Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestName() {
	s.route.Get("/name/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().Name(),
		})
	}).Name("test-name")

	req, err := http.NewRequest("GET", "/name/1", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)
	s.Equal("{\"name\":\"test-name\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInfo() {
	s.route.Get("/info/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"info": ctx.Request().Info(),
		})
	}).Name("test-info-get")
	s.route.Post("/info/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"info": ctx.Request().Info(),
		})
	}).Name("test-info-post")
	s.route.Any("/info/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"info": ctx.Request().Info(),
		})
	}).Name("test-info-any")
	s.route.Any("/info/resource/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"info": ctx.Request().Info(),
		})
	}).Name("test-info-resource")

	req, err := http.NewRequest("GET", "/info/1", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)
	s.Equal(`{"info":{"handler":"github.com/goravel/fiber.(*ContextRequestSuite).TestInfo.func1","method":"GET","name":"test-info-get","path":"/info/{id}"}}`, body)
	s.Equal(http.StatusOK, code)

	req, err = http.NewRequest("HEAD", "/info/1", nil)
	s.Require().Nil(err)

	code, body, _, _ = s.request(req)
	s.Empty(body)
	s.Equal(http.StatusOK, code)

	req, err = http.NewRequest("POST", "/info/1", nil)
	s.Require().Nil(err)

	code, body, _, _ = s.request(req)
	s.Equal(`{"info":{"handler":"github.com/goravel/fiber.(*ContextRequestSuite).TestInfo.func2","method":"POST","name":"test-info-post","path":"/info/{id}"}}`, body)
	s.Equal(http.StatusOK, code)

	req, err = http.NewRequest("GET", "/info/any/1", nil)
	s.Require().Nil(err)

	code, body, _, _ = s.request(req)
	s.Equal(`{"info":{"handler":"github.com/goravel/fiber.(*ContextRequestSuite).TestInfo.func3","method":"GET","name":"test-info-any","path":"/info/any/{id}"}}`, body)
	s.Equal(http.StatusOK, code)

	req, err = http.NewRequest("POST", "/info/resource/1", nil)
	s.Require().Nil(err)

	code, body, _, _ = s.request(req)
	s.Equal(`{"info":{"handler":"github.com/goravel/fiber.(*ContextRequestSuite).TestInfo.func4","method":"POST","name":"test-info-resource","path":"/info/resource/{id}"}}`, body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_Json() {
	s.route.Post("/input/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":     ctx.Request().Input("id"),
			"int":    ctx.Request().Input("int"),
			"map":    ctx.Request().Input("map"),
			"string": ctx.Request().Input("string"),
		})
	})

	payload := strings.NewReader(`{
		"id": "3",
		"string": ["string 1", "string 2"],
		"int": [1, 2],
		"map": {"a": "b"}
	}`)
	req, err := http.NewRequest("POST", "/input/json/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"3\",\"int\":\"1,2\",\"map\":\"{\\\"a\\\":\\\"b\\\"}\",\"string\":\"string 1,string 2\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_Form() {
	s.route.Post("/input/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":     ctx.Request().Input("id"),
			"map":    ctx.Request().Input("map"),
			"string": ctx.Request().Input("string[]"),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.WriteField("id", "4")
	s.Require().Nil(err)

	err = writer.WriteField("string[]", "string 1")
	s.Require().Nil(err)

	err = writer.WriteField("string[]", "string 2")
	s.Require().Nil(err)

	err = writer.WriteField("map", "{\"a\":\"b\"}")
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/input/form/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"4\",\"map\":\"{\\\"a\\\":\\\"b\\\"}\",\"string\":\"string 1,string 2\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_Url() {
	s.route.Post("/input/url/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":     ctx.Request().Input("id"),
			"map":    ctx.Request().Input("map"),
			"string": ctx.Request().Input("string"),
		})
	})

	payload := neturl.Values{
		"id":     {"4"},
		"map":    {"{\"a\":\"b\"}"},
		"string": {"string 1", "string 2"},
	}
	req, err := http.NewRequest("POST", "/input/url/1?id=2", strings.NewReader(payload.Encode()))
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"4\",\"map\":\"{\\\"a\\\":\\\"b\\\"}\",\"string\":\"string 1,string 2\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_Route() {
	s.route.Post("/input/route/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	req, err := http.NewRequest("POST", "/input/route/1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"1\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_KeyInBodyIsEmpty() {
	s.route.Post("/input/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id1": ctx.Request().Input("id1", "a"),
		})
	})

	payload := strings.NewReader(`{
		"id1": ""
	}`)
	req, err := http.NewRequest("POST", "/input/empty/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id1\":\"\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_KeyInQueryIsEmpty() {
	s.route.Post("/input/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id1": ctx.Request().Input("id1", "a"),
		})
	})

	req, err := http.NewRequest("POST", "/input/empty/1?id1=", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id1\":\"\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_Default() {
	s.route.Post("/input/default/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id1": ctx.Request().Input("id1", "a"),
		})
	})

	req, err := http.NewRequest("POST", "/input/default/1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id1\":\"a\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_NestedJson() {
	s.route.Post("/input/nested/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":      ctx.Request().Input("id.a"),
			"string0": ctx.Request().Input("string.0"),
			"string":  ctx.Request().Input("string"),
		})
	})

	payload := strings.NewReader(`{
		"id": {"a": {"b": "c"}},
		"string": ["string 0", "string 1"]
	}`)
	req, err := http.NewRequest("POST", "/input/nested/json/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"{\\\"b\\\":\\\"c\\\"}\",\"string\":\"string 0,string 1\",\"string0\":\"string 0\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_NestedForm() {
	s.route.Post("/input/nested/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name":    ctx.Request().Input("name"),
			"string0": ctx.Request().Input("string.0"),
			"string":  ctx.Request().Input("string"),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.WriteField("name", "goravel")
	s.Require().Nil(err)

	err = writer.WriteField("string[]", "string 0")
	s.Require().Nil(err)

	err = writer.WriteField("string[]", "string 1")
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/input/nested/form/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"goravel\",\"string\":\"string 0,string 1\",\"string0\":\"string 0\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInput_NestedUrl() {
	s.route.Post("/input/nested/url/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":      ctx.Request().Input("id"),
			"string0": ctx.Request().Input("string.0"),
			"string":  ctx.Request().Input("string"),
		})
	})

	form := neturl.Values{
		"id":     {"4"},
		"string": {"string 0", "string 1"},
	}
	req, err := http.NewRequest("POST", "/input/nested/url/1?id=2", strings.NewReader(form.Encode()))
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":\"4\",\"string\":\"string 0,string 1\",\"string0\":\"string 0\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_Default() {
	s.route.Post("/input-array/default/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputArray("name", []string{"a", "b"}),
		})
	})

	payload := strings.NewReader(`{
		"id": ["id 0", "id 1"]
	}`)
	req, err := http.NewRequest("POST", "/input-array/default/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":[\"a\",\"b\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_KeyInBodyIsEmpty() {
	s.route.Post("/input-array/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputArray("name", []string{"a", "b"}),
		})
	})

	payload := strings.NewReader(`{
		"name": []
	}`)
	req, err := http.NewRequest("POST", "/input-array/empty/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":[]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_KeyInQueryIsEmpty() {
	s.route.Post("/input-array/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputArray("name", []string{"a", "b"}),
		})
	})

	req, err := http.NewRequest("POST", "/input-array/empty/1?name=", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":[]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_Json() {
	s.route.Post("/input-array/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputArray("id"),
		})
	})

	payload := strings.NewReader(`{
		"id": ["id 0", "id 1"]
	}`)
	req, err := http.NewRequest("POST", "/input-array/json/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":[\"id 0\",\"id 1\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_Form() {
	s.route.Post("/input-array/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"InputArray":  ctx.Request().InputArray("arr[]"),
			"InputArray1": ctx.Request().InputArray("arr"),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.WriteField("arr[]", "arr 1")
	s.Require().Nil(err)

	err = writer.WriteField("arr[]", "arr 2")
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/input-array/form/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"InputArray\":[\"arr 1\",\"arr 2\"],\"InputArray1\":[\"arr 1\",\"arr 2\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputArray_Url() {
	s.route.Post("/input-array/url/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"string":  ctx.Request().InputArray("string[]"),
			"string1": ctx.Request().InputArray("string"),
		})
	})

	form := neturl.Values{
		"string[]": {"string 0", "string 1"},
	}
	req, err := http.NewRequest("POST", "/input-array/url/1?id=2", strings.NewReader(form.Encode()))
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body, _, _ := s.request(req)

	s.Equal("{\"string\":[\"string 0\",\"string 1\"],\"string1\":[\"string 0\",\"string 1\"]}", body)
	s.Equal(http.StatusOK, code)
}

// Test Issue: https://github.com/goravel/goravel/issues/659
func (s *ContextRequestSuite) TestInputArray_UrlMore() {
	s.route.Post("/input-array/url/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"string":  ctx.Request().InputArray("string[]"),
			"string1": ctx.Request().InputArray("string"),
		})
	})

	form := neturl.Values{
		"string[]": {"string 0", "string 1", "string 2", "string 3"},
	}
	req, err := http.NewRequest("POST", "/input-array/url/1?id=2", strings.NewReader(form.Encode()))
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body, _, _ := s.request(req)

	s.Equal("{\"string\":[\"string 0\",\"string 1\",\"string 2\",\"string 3\"],\"string1\":[\"string 0\",\"string 1\",\"string 2\",\"string 3\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_Default() {
	s.route.Post("/input-map/default/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputMap("name", map[string]any{"a": "b"}),
		})
	})

	payload := strings.NewReader(`{
		"id": {"a": "3", "b": "4"}
	}`)
	req, err := http.NewRequest("POST", "/input-map/default/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":{\"a\":\"b\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_KeyInBodyIsEmpty() {
	s.route.Post("/input-map/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputMap("name", map[string]any{
				"a": "b",
			}),
		})
	})

	payload := strings.NewReader(`{
		"name": {}
	}`)
	req, err := http.NewRequest("POST", "/input-map/empty/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":{}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_KeyInQueryIsEmpty() {
	s.route.Post("/input-map/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().InputMap("name", map[string]any{
				"a": "b",
			}),
		})
	})

	req, err := http.NewRequest("POST", "/input-map/empty/1?name=", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":{}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_Json() {
	s.route.Post("/input-map/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputMap("id"),
		})
	})

	payload := strings.NewReader(`{
		"id": {"a": "3", "b": "4"}
	}`)
	req, err := http.NewRequest("POST", "/input-map/json/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":{\"a\":\"3\",\"b\":\"4\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_Form() {
	s.route.Post("/input-map/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputMap("id"),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.WriteField("id", "{\"a\":\"3\",\"b\":\"4\"}")
	s.Require().Nil(err)

	err = writer.Close()
	s.Require().Nil(err)

	req, err := http.NewRequest("POST", "/input-map/form/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":{\"a\":\"3\",\"b\":\"4\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMap_Url() {
	s.route.Post("/input-map/url/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputMap("id"),
		})
	})

	form := neturl.Values{
		"id": {"{\"a\":\"3\",\"b\":\"4\"}"},
	}
	req, err := http.NewRequest("POST", "/input-map/url/1?id=2", strings.NewReader(form.Encode()))
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":{\"a\":\"3\",\"b\":\"4\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMapArray_Default() {
	s.route.Post("/input-map-array/default/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"names": ctx.Request().InputMapArray("names", []map[string]any{{"a": "b"}}),
		})
	})

	payload := strings.NewReader(`{}`)
	req, err := http.NewRequest("POST", "/input-map-array/default/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"names\":[{\"a\":\"b\"}]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMapArray_KeyInBodyIsEmpty() {
	s.route.Post("/input-map-array/empty/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"names": ctx.Request().InputMapArray("names", []map[string]any{
				{"a": "b"},
			}),
		})
	})

	payload := strings.NewReader(`{
		"names": []
	}`)
	req, err := http.NewRequest("POST", "/input-map-array/empty/1", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"names\":[]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMapArray_Json() {
	s.route.Post("/input-map-array/json/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"ids": ctx.Request().InputMapArray("ids"),
		})
	})

	payload := strings.NewReader(`{
		"ids": [{"a": "3"},{"b": "4"}]
	}`)
	req, err := http.NewRequest("POST", "/input-map-array/json/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"ids\":[{\"a\":\"3\"},{\"b\":\"4\"}]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputMapArray_Form() {
	s.route.Post("/input-map-array/form/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"ids": ctx.Request().InputMapArray("ids"),
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	s.Require().Nil(writer.WriteField("ids", `{"a":"3"}`))
	s.Require().Nil(writer.WriteField("ids", `{"b":"4"}`))
	s.Require().Nil(writer.Close())

	req, err := http.NewRequest("POST", "/input-map-array/form/1?id=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"ids\":[{\"a\":\"3\"},{\"b\":\"4\"}]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputInt() {
	s.route.Post("/input-int/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputInt("id"),
		})
	})

	req, err := http.NewRequest("POST", "/input-int/1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":1}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputInt64() {
	s.route.Post("/input-int64/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().InputInt64("id"),
		})
	})

	req, err := http.NewRequest("POST", "/input-int64/1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id\":1}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestInputBool() {
	s.route.Post("/input-bool/{id1}/{id2}/{id3}/{id4}/{id5}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id1": ctx.Request().InputBool("id1"),
			"id2": ctx.Request().InputBool("id2"),
			"id3": ctx.Request().InputBool("id3"),
			"id4": ctx.Request().InputBool("id4"),
			"id5": ctx.Request().InputBool("id5"),
		})
	})

	req, err := http.NewRequest("POST", "/input-bool/1/true/on/yes/a", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"id1\":true,\"id2\":true,\"id3\":true,\"id4\":true,\"id5\":false}", body)
	s.Equal(http.StatusOK, code)
}

// Test Issue: https://github.com/goravel/goravel/issues/528
func (s *ContextRequestSuite) TestPostWithEmpty() {
	s.route.Post("/post-with-empty", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().All(),
		})
	})

	payload := strings.NewReader("")
	req, err := http.NewRequest("POST", "/post-with-empty?a=1", payload)
	req.ContentLength = 1
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"a\":\"1\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestQuery() {
	s.route.Get("/query", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"string":        ctx.Request().Query("string", ""),
			"int":           ctx.Request().QueryInt("int", 11),
			"int_default":   ctx.Request().QueryInt("int_default", 11),
			"int64":         ctx.Request().QueryInt64("int64", 22),
			"int64_default": ctx.Request().QueryInt64("int64_default", 22),
			"bool1":         ctx.Request().QueryBool("bool1"),
			"bool2":         ctx.Request().QueryBool("bool2"),
			"bool3":         ctx.Request().QueryBool("bool3"),
			"bool4":         ctx.Request().QueryBool("bool4"),
			"bool5":         ctx.Request().QueryBool("bool5"),
			"error":         ctx.Request().QueryInt("error", 33),
		})
	})

	req, err := http.NewRequest("GET", "/query?string=Goravel&int=1&int64=2&bool1=1&bool2=true&bool3=on&bool4=yes&bool5=0&error=a", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"bool1\":true,\"bool2\":true,\"bool3\":true,\"bool4\":true,\"bool5\":false,\"error\":0,\"int\":1,\"int64\":2,\"int64_default\":22,\"int_default\":11,\"string\":\"Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestQueryArray() {
	s.route.Get("/query-array", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().QueryArray("name"),
		})
	})

	req, err := http.NewRequest("GET", "/query-array?name=Goravel&name=Goravel1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":[\"Goravel\",\"Goravel1\"]}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestQueryMap() {
	s.route.Get("/query-map", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"name": ctx.Request().QueryMap("name"),
		})
	})

	req, err := http.NewRequest("GET", "/query-map?name[a]=Goravel&name[b]=Goravel1", nil)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":{\"a\":\"Goravel\",\"b\":\"Goravel1\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestQueries() {
	s.route.Post("/queries", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"all": ctx.Request().Queries(),
		})
	})

	payload := strings.NewReader(`{
		"string": "error"
	}`)
	req, err := http.NewRequest("POST", "/queries?string=Goravel&int=1&int64=2&bool1=1&bool2=true&bool3=on&bool4=yes&bool5=0&error=a", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"all\":{\"bool1\":\"1\",\"bool2\":\"true\",\"bool3\":\"on\",\"bool4\":\"yes\",\"bool5\":\"0\",\"error\":\"a\",\"int\":\"1\",\"int64\":\"2\",\"string\":\"Goravel\"}}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestRoute() {
	s.route.Get("/route/{string}/{int}/{int64}/{string1}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"string": ctx.Request().Route("string"),
			"int":    ctx.Request().RouteInt("int"),
			"int64":  ctx.Request().RouteInt64("int64"),
			"error":  ctx.Request().RouteInt("string1"),
		})
	})

	req, err := http.NewRequest("GET", "/route/1/2/3/a", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"error\":0,\"int\":2,\"int64\":3,\"string\":\"1\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestSession() {
	s.route.Get("/session", func(ctx contractshttp.Context) contractshttp.Response {
		ctx.Request().SetSession(session.NewSession("goravel_session", nil, foundationjson.New()))

		return ctx.Response().Success().Json(contractshttp.Json{
			"message": ctx.Request().Session().GetName(),
		})
	})

	req, err := http.NewRequest("GET", "/session", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"message\":\"goravel_session\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestSession_NotSet() {
	s.route.Get("/session/not-set", func(ctx contractshttp.Context) contractshttp.Response {
		if ctx.Request().HasSession() {
			return ctx.Response().Success().Json(contractshttp.Json{
				"message": ctx.Request().Session().GetName(),
			})
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"message": "session not set",
		})
	})

	req, err := http.NewRequest("GET", "/session/not-set", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"message\":\"session not set\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidate_GetSuccess() {
	s.route.Get("/validate/get-success/{uuid}", func(ctx contractshttp.Context) contractshttp.Response {
		validator, err := ctx.Request().Validate(map[string]string{
			"uuid": "min_len:2",
			"name": "required",
		}, validation.Filters(map[string]string{
			"uuid": "trim",
			"name": "trim",
		}))
		if err != nil {
			return ctx.Response().String(400, "Validate error: "+err.Error())
		}
		if validator.Fails() {
			return ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
		}

		type Test struct {
			Uuid string `form:"uuid" json:"uuid"`
			Name string `form:"name" json:"name"`
		}
		var test Test
		if err := validator.Bind(&test); err != nil {
			return ctx.Response().String(400, "Validate bind error: "+err.Error())
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"uuid": test.Uuid,
			"name": test.Name,
		})
	})

	req, err := http.NewRequest("GET", "/validate/get-success/abc?name=Goravel", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel\",\"uuid\":\"abc\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidate_GetFail() {
	s.route.Get("/validate/get-fail/{uuid}", func(ctx contractshttp.Context) contractshttp.Response {
		validator, err := ctx.Request().Validate(map[string]string{
			"uuid": "min_len:4",
			"name": "required",
		}, validation.Filters(map[string]string{
			"uuid": "trim",
			"name": "trim",
		}))
		if err != nil {
			return ctx.Response().String(400, "Validate error: "+err.Error())
		}
		if validator.Fails() {
			return ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
		}

		return nil
	})

	req, err := http.NewRequest("GET", "/validate/get-fail/abc?name=Goravel", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("Validate fail: map[uuid:map[min_len:uuid min length is 4]]", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) TestValidate_PostSuccess() {
	s.route.Post("/validate/post-success/{id}/{uuid}", func(ctx contractshttp.Context) contractshttp.Response {
		validator, err := ctx.Request().Validate(map[string]string{
			"id":   "required",
			"uuid": "required",
			"age":  "required",
			"name": "required",
		}, validation.Filters(map[string]string{
			"id":   "trim",
			"uuid": "trim",
			"age":  "trim",
			"name": "trim",
		}))
		if err != nil {
			return ctx.Response().String(400, "Validate error: "+err.Error())
		}
		if validator.Fails() {
			return ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
		}

		type Test struct {
			ID   string `form:"id" json:"id"`
			Uuid string `form:"uuid" json:"uuid"`
			Age  string `form:"age" json:"age"`
			Name string `form:"name" json:"name"`
		}
		var test Test
		if err := validator.Bind(&test); err != nil {
			return ctx.Response().String(400, "Validate bind error: "+err.Error())
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"id":   test.ID,
			"uuid": test.Uuid,
			"age":  test.Age,
			"name": test.Name,
		})
	})

	payload := strings.NewReader(`{
		"name": " Goravel ",
		"uuid": " 3 "
	}`)
	req, err := http.NewRequest("POST", "/validate/post-success/1/2?age=2", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"age\":\"2\",\"id\":\"1\",\"name\":\"Goravel\",\"uuid\":\"3\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidate_PostFail() {
	s.route.Post("/validate/post-fail", func(ctx contractshttp.Context) contractshttp.Response {
		validator, err := ctx.Request().Validate(map[string]string{
			"name1": "required",
		}, validation.Filters(map[string]string{
			"name1": "trim",
		}))
		if err != nil {
			return ctx.Response().String(400, "Validate error: "+err.Error())
		}
		if validator.Fails() {
			return ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": "",
		})
	})

	payload := strings.NewReader(`{
		"name": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/validate/post-fail", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("Validate fail: map[name1:map[required:name1 is required to not be empty]]", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) TestValidateRequest_PrepareForValidationWithContext() {
	s.route.Get("/validate-request/prepare-for-validation-with-context", func(ctx contractshttp.Context) contractshttp.Response {
		// nolint:all
		ctx.WithValue("test", "-ctx")

		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	req, err := http.NewRequest("GET", "/validate-request/prepare-for-validation-with-context?name=Goravel", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel1-ctx\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidateRequest_GetSuccess() {
	s.route.Get("/validate-request/get-success", func(ctx contractshttp.Context) contractshttp.Response {
		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	req, err := http.NewRequest("GET", "/validate-request/get-success?name=Goravel", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel1\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidateRequest_GetFail() {
	s.route.Get("/validate-request/get-fail", func(ctx contractshttp.Context) contractshttp.Response {
		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	req, err := http.NewRequest("GET", "/validate-request/get-fail?name1=Goravel", nil)
	s.Require().Nil(err)

	code, body, _, _ := s.request(req)

	s.Equal("Validate fail: map[name:map[required:name is required to not be empty]]", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) TestValidateRequest_FormSuccess() {
	s.route.Post("/validate-request/form-success", func(ctx contractshttp.Context) contractshttp.Response {
		var request FileImageJson
		validateErrors, err := ctx.Request().ValidateRequest(&request)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name":  request.Name,
			"file":  request.File.Filename,
			"image": request.Image.Filename,
			"json":  request.Json,
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	err := writer.WriteField("name", "Goravel")
	s.Require().NoError(err)

	err = writer.WriteField("json", `{"age": 1, "name": "Bowen"}`)
	s.Require().NoError(err)

	readme, err := os.Open("./README.md")
	s.Require().NoError(err)
	defer func() {
		_ = readme.Close()
	}()

	formFile, err := writer.CreateFormFile("file", filepath.Base("./README.md"))
	s.Require().NoError(err)

	_, err = io.Copy(formFile, readme)
	s.Require().NoError(err)

	logo, err := os.Open("./logo.png")
	s.Require().NoError(err)
	defer func() {
		_ = logo.Close()
	}()

	formImage, err := writer.CreateFormFile("image", filepath.Base("./logo.png"))
	s.Require().NoError(err)

	_, err = io.Copy(formImage, logo)
	s.Require().NoError(err)

	err = writer.Close()
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", "/validate-request/form-success", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("{\"file\":\"README.md\",\"image\":\"logo.png\",\"json\":\"{\\\"age\\\": 1, \\\"name\\\": \\\"Bowen\\\"}\",\"name\":\"Goravel\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidateRequest_FormFail() {
	s.route.Post("/validate-request/form-success", func(ctx contractshttp.Context) contractshttp.Response {
		var request FileImageJson
		validateErrors, err := ctx.Request().ValidateRequest(&request)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name":  request.Name,
			"file":  request.File.Filename,
			"image": request.Image.Filename,
			"json":  request.Json,
		})
	})

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	err := writer.WriteField("name", "Goravel")
	s.Require().NoError(err)

	err = writer.WriteField("json", `{"age": 1, "name": "Bowen",}`)
	s.Require().NoError(err)

	readme, err := os.Open("./README.md")
	s.Require().NoError(err)
	defer func() {
		_ = readme.Close()
	}()

	formFile, err := writer.CreateFormFile("file", filepath.Base("./README.md"))
	s.Require().NoError(err)

	_, err = io.Copy(formFile, readme)
	s.Require().NoError(err)

	logo, err := os.Open("./README.md")
	s.Require().NoError(err)
	defer func() {
		_ = logo.Close()
	}()

	formImage, err := writer.CreateFormFile("image", filepath.Base("./README.md"))
	s.Require().NoError(err)

	_, err = io.Copy(formImage, logo)
	s.Require().NoError(err)

	err = writer.Close()
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", "/validate-request/form-success", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	code, body, _, _ := s.request(req)

	s.Equal("Validate fail: map[image:map[image:image value must be an image] json:map[json:json value should be a json string]]", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) TestValidateRequest_JsonSuccess() {
	s.route.Post("/validate-request/json-success", func(ctx contractshttp.Context) contractshttp.Response {
		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	payload := strings.NewReader(`{
		"name": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/validate-request/json-success", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel1\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidateRequest_JsonFail() {
	s.route.Post("/validate-request/json-fail", func(ctx contractshttp.Context) contractshttp.Response {
		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	payload := strings.NewReader(`{
		"name1": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/validate-request/json-fail", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("Validate fail: map[name:map[required:name is required to not be empty]]", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) TestValidateRequest_PostSuccessWithFilter() {
	s.route.Post("/validate-request/filter/post-success", func(ctx contractshttp.Context) contractshttp.Response {
		var createUser CreateUser
		validateErrors, err := ctx.Request().ValidateRequest(&createUser)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": createUser.Name,
		})
	})

	payload := strings.NewReader(`{
		"name": " Goravel "
	}`)
	req, err := http.NewRequest("POST", "/validate-request/filter/post-success", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("{\"name\":\"Goravel 1\"}", body)
	s.Equal(http.StatusOK, code)
}

func (s *ContextRequestSuite) TestValidateRequest_Unauthorize() {
	s.route.Post("/validate-request/unauthorize", func(ctx contractshttp.Context) contractshttp.Response {
		var unauthorize Unauthorize
		validateErrors, err := ctx.Request().ValidateRequest(&unauthorize)
		if err != nil {
			return ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
		}
		if validateErrors != nil {
			return ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
		}

		return ctx.Response().Success().Json(contractshttp.Json{
			"name": unauthorize.Name,
		})
	})

	payload := strings.NewReader(`{
		"name": "Goravel"
	}`)
	req, err := http.NewRequest("POST", "/validate-request/unauthorize", payload)
	s.Require().Nil(err)

	req.Header.Set("Content-Type", "application/json")
	code, body, _, _ := s.request(req)

	s.Equal("Validate error: error", body)
	s.Equal(http.StatusBadRequest, code)
}

func (s *ContextRequestSuite) request(req *http.Request) (int, string, http.Header, []*http.Cookie) {
	resp, err := s.route.Test(req)
	s.NoError(err)

	body, err := io.ReadAll(resp.Body)
	s.NoError(err)

	return resp.StatusCode, string(body), resp.Header, resp.Cookies()
}

func TestGetValueFromHttpBody(t *testing.T) {
	tests := []struct {
		name        string
		httpBody    map[string]any
		key         string
		expectValue any
	}{
		{
			name: "Return nil when httpBody is nil",
		},
		{
			name:        "Return string when httpBody is map[string]string",
			httpBody:    map[string]any{"name": "goravel"},
			key:         "name",
			expectValue: "goravel",
		},
		{
			name:        "Return map when httpBody is map[string]map[string]string",
			httpBody:    map[string]any{"name": map[string]string{"sub": "goravel"}},
			key:         "name",
			expectValue: map[string]string{"sub": "goravel"},
		},
		{
			name:        "Return slice when httpBody is map[string][]string",
			httpBody:    map[string]any{"name[]": []string{"a", "b"}},
			key:         "name[]",
			expectValue: []string{"a", "b"},
		},
		{
			name:        "Return slice when httpBody is map[string][]string, but key doesn't contain []",
			httpBody:    map[string]any{"name": []string{"a", "b"}},
			key:         "name",
			expectValue: []string{"a", "b"},
		},
		{
			name:        "Return string when httpBody is map[string]map[string]string and key with point",
			httpBody:    map[string]any{"name": map[string]string{"sub": "goravel"}},
			key:         "name.sub",
			expectValue: "goravel",
		},
		{
			name:        "Return int when httpBody is map[string]map[string]int and key with point",
			httpBody:    map[string]any{"name": map[string]int{"sub": 1}},
			key:         "name.sub",
			expectValue: 1,
		},
		{
			name:        "Return string when httpBody is map[string][]string and key with point",
			httpBody:    map[string]any{"name[]": []string{"a", "b"}},
			key:         "name[].0",
			expectValue: "a",
		},
		{
			name:        "Return string when httpBody is map[string][]string and key with point and index is 1",
			httpBody:    map[string]any{"name[]": []string{"a", "b"}},
			key:         "name[].1",
			expectValue: "b",
		},
		{
			name:        "Return string when httpBody is map[string][]string and key with point, but key doesn't contain []",
			httpBody:    map[string]any{"name[]": []string{"a", "b"}},
			key:         "name.0",
			expectValue: "a",
		},
		{
			name:        "Return string when httpBody is map[string][]string and key with point and index is 1, but key doesn't contain []",
			httpBody:    map[string]any{"name[]": []string{"a", "b"}},
			key:         "name.1",
			expectValue: "b",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contextRequest := &ContextRequest{
				httpBody: test.httpBody,
			}

			value := contextRequest.getValueFromHttpBody(test.key)
			assert.Equal(t, test.expectValue, value)
		})
	}
}

// Timeout creates middleware to set a timeout for a request
func testAllMiddleware() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		all := ctx.Request().All()
		ctx.WithValue("all", all)

		ctx.Request().Next()
	}
}
