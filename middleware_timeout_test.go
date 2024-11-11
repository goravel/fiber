package fiber

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTimeoutMiddleware(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()

	route, err := NewRoute(mockConfig, nil)
	require.NoError(t, err)

	route.Middleware(Timeout(1*time.Second)).Get("/timeout", func(ctx contractshttp.Context) contractshttp.Response {
		time.Sleep(2 * time.Second)

		return ctx.Response().Success().String("timeout")
	})
	route.Middleware(Timeout(1*time.Second)).Get("/normal", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().String("normal")
	})
	route.Middleware(Timeout(1*time.Second)).Get("/panic", func(ctx contractshttp.Context) contractshttp.Response {
		panic(1)
	})

	req, err := http.NewRequest("GET", "/timeout", nil)
	require.NoError(t, err)

	resp, err := route.instance.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	req, err = http.NewRequest("GET", "/normal", nil)
	require.NoError(t, err)

	resp, err = route.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "normal", string(body))

	req, err = http.NewRequest("GET", "/panic", nil)
	require.NoError(t, err)

	mockLog := mockslog.NewLog(t)
	mockLog.EXPECT().Request(mock.Anything).Return(mockLog).Once()
	mockLog.EXPECT().Error(mock.Anything).Once()
	LogFacade = mockLog

	resp, err = route.instance.Test(req)
	fmt.Printf("resp: %+v\n", resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}
