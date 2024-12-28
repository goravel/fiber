package fiber

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
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
		return nil
	})

	route.Middleware(Timeout(1*time.Second)).Get("/normal", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().String("normal")
	})

	route.Middleware(Timeout(5*time.Second)).Get("/panic", func(ctx contractshttp.Context) contractshttp.Response {
		panic("test panic")
	})

	globalRecover := func(ctx contractshttp.Context, err any) {
		ctx.Request().AbortWithStatusJson(http.StatusInternalServerError, fiber.Map{"error": "Internal Panic"})
	}
	route.Recover(globalRecover)

	mockLog := mockslog.NewLog(t)
	mockLog.On("Error", mock.Anything).Return(nil)
	mockLog.On("Info", "Request completed normally").Return(nil)
	mockLog.On("Error", "Timeout occurred").Return(nil)
	LogFacade = mockLog

	t.Run("timeout", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/timeout", nil)
		require.NoError(t, err)

		resp, err := route.instance.Test(req, -1)
		require.NoError(t, err)
		require.NotNil(t, resp) // Проверяем, что ответ не nil

		assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"error":"Request Timeout"}`, string(body))
	})

	t.Run("normal", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/normal", nil)
		require.NoError(t, err)

		resp, err := route.instance.Test(req, -1)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "normal", string(body))
	})

	t.Run("panic", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/panic", nil)
		require.NoError(t, err)

		resp, err := route.instance.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"error":"Internal Panic"}`, string(body))
	})
}
