package fiber

import (
	"crypto/tls"
	"net/http"
	"testing"

	configmocks "github.com/goravel/framework/contracts/config/mocks"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestTls(t *testing.T) {
	var (
		mockConfig *configmocks.Config
		resp       *http.Response
	)
	beforeEach := func() {
		mockConfig = &configmocks.Config{}
	}

	tests := []struct {
		name  string
		setup func()
	}{
		{
			name: "not use tls",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{}).Once()
				mockConfig.On("GetString", "http.tls.host").Return("").Once()
				mockConfig.On("GetString", "http.tls.port").Return("").Once()
				mockConfig.On("GetString", "http.tls.ssl.cert").Return("").Once()
				mockConfig.On("GetString", "http.tls.ssl.key").Return("").Once()
				ConfigFacade = mockConfig
			},
		},
		{
			name: "use tls",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{}).Once()
				mockConfig.On("GetString", "http.tls.host").Return("127.0.0.1").Once()
				mockConfig.On("GetString", "http.tls.port").Return("3000").Once()
				mockConfig.On("GetString", "http.tls.ssl.cert").Return("test_ca.crt").Once()
				mockConfig.On("GetString", "http.tls.ssl.key").Return("test_ca.key").Once()
				ConfigFacade = mockConfig
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			test.setup()

			f := NewRoute(mockConfig)
			f.GlobalMiddleware()
			f.Any("/any/{id}", func(ctx contractshttp.Context) {
				ctx.Response().Success().Json(contractshttp.Json{
					"id": ctx.Request().Input("id"),
				})
			})

			req, err := http.NewRequest("POST", "/any/1", nil)
			req.TLS = &tls.ConnectionState{}
			assert.Nil(t, err)

			resp, err = f.Test(req)
			assert.NoError(t, err, test.name)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			mockConfig.AssertExpectations(t)
		})
	}
}
