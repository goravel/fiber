package fiber

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	contractshttp "github.com/goravel/framework/contracts/http"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTimeoutMiddleware(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
	mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()

	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	err := route.init(nil)
	require.Nil(t, err)

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

	route.Middleware(Timeout(1*time.Second)).Get("/panic-after-timeout", func(ctx contractshttp.Context) contractshttp.Response {
		time.Sleep(2 * time.Second)
		panic("panic after timeout")
	})

	t.Run("timeout", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/timeout", nil)
		require.NoError(t, err)
		req.Host = "example.com"
		resp, err := route.instance.Test(req, fiber.TestConfig{Timeout: 0})
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, contractshttp.StatusRequestTimeout, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "Request Timeout", string(body))
	})

	t.Run("normal", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/normal", nil)
		require.NoError(t, err)
		req.Host = "example.com"
		resp, err := route.instance.Test(req, fiber.TestConfig{Timeout: 0})
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "normal", string(body))
	})

	t.Run("panic with default recover", func(t *testing.T) {
		mockLog := mockslog.NewLog(t)
		mockLog.EXPECT().WithContext(mock.Anything).Return(mockLog).Once()
		mockLog.EXPECT().Request(mock.Anything).Return(mockLog).Once()
		mockLog.EXPECT().Error("test panic").Once()
		LogFacade = mockLog

		req, err := http.NewRequest("GET", "/panic", nil)
		require.NoError(t, err)
		req.Host = "example.com"
		resp, err := route.instance.Test(req, fiber.TestConfig{Timeout: 0})
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "Internal Server Error", string(body))
	})

	t.Run("panic with custom recover", func(t *testing.T) {
		mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
		mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()

		globalRecover := func(ctx contractshttp.Context, err any) {
			ctx.Request().Abort(http.StatusBadRequest)
		}
		route.Recover(globalRecover)

		route.Middleware(Timeout(5*time.Second)).Get("/panic", func(ctx contractshttp.Context) contractshttp.Response {
			panic("test panic")
		})

		req, err := http.NewRequest("GET", "/panic", nil)
		require.NoError(t, err)
		req.Host = "example.com"
		resp, err := route.instance.Test(req, fiber.TestConfig{Timeout: 0})
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "Bad Request", string(body))

		globalRecoverCallback = defaultRecoverCallback
	})

	t.Run("panic after timeout should not panic on recycled context", func(t *testing.T) {
		mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
		mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()

		globalRecoverCallback = defaultRecoverCallback
		err := route.init(nil)
		require.NoError(t, err)

		route.Middleware(Timeout(1 * time.Second)).Get("/panic-after-timeout", func(ctx contractshttp.Context) contractshttp.Response {
			time.Sleep(2 * time.Second)
			panic("panic after timeout")
		})

		req, err := http.NewRequest("GET", "/panic-after-timeout", nil)
		require.NoError(t, err)

		resp, err := route.instance.Test(req, -1)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, contractshttp.StatusRequestTimeout, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "Request Timeout", string(body))

		// Wait for the background goroutine to finish panicking and recovering
		// without crashing the process.
		time.Sleep(3 * time.Second)
	})
}
