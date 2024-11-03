package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	mocksconfig "github.com/goravel/framework/mocks/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutMiddleware(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
	mockConfig.EXPECT().GetInt("http.drivers.gin.body_limit", 4096).Return(4096).Once()

	route, err := NewRoute(mockConfig, nil)
	require.NoError(t, err)

	route.Middleware(Timeout(1*time.Second)).Get("/timeout", func(ctx contractshttp.Context) contractshttp.Response {
		time.Sleep(2 * time.Second)

		return ctx.Response().Success().String("timeout")
	})
	route.Middleware(Timeout(1*time.Second)).Get("/normal", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().String("normal")
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/timeout", nil)
	require.NoError(t, err)

	route.ServeHTTP(w, req)
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/normal", nil)
	require.NoError(t, err)

	route.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "normal", w.Body.String())
}
