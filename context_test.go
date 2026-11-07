package fiber

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	type customType struct{}
	var customTypeKey customType

	httpCtx := Background()
	httpCtx.WithValue("Hello", "world")
	httpCtx.WithValue("Hi", "Goravel")
	httpCtx.WithValue(customTypeKey, "halo")
	httpCtx.WithValue(sessionKey, "session")

	userContext := context.Background()
	//nolint:all
	userContext = context.WithValue(userContext, "user_a", "b")
	httpCtx.WithContext(userContext)

	httpCtx.WithValue(1, "one")
	httpCtx.WithValue(2.2, "two point two")

	assert.Equal(t, "world", httpCtx.Value("Hello"))
	assert.Equal(t, "Goravel", httpCtx.Value("Hi"))
	assert.Equal(t, "halo", httpCtx.Value(customTypeKey))
	assert.Equal(t, "one", httpCtx.Value(1))
	assert.Equal(t, "two point two", httpCtx.Value(2.2))
	assert.Equal(t, "session", httpCtx.Value(sessionKey))

	// The value of UserContext can't be covered.
	assert.Equal(t, "b", httpCtx.Value("user_a"))

	ctx := httpCtx.Context()
	assert.Equal(t, "world", ctx.Value("Hello"))
	assert.Equal(t, "Goravel", ctx.Value("Hi"))
	assert.Equal(t, "halo", ctx.Value(customTypeKey))
	assert.Equal(t, "b", ctx.Value("user_a"))
	assert.Equal(t, "one", ctx.Value(1))
	assert.Equal(t, "two point two", ctx.Value(2.2))
	assert.Nil(t, ctx.Value(sessionKey))
}

func TestContext_Concurrent(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (writer, reader *Context)
	}{
		{
			name: "same context",
			setup: func(t *testing.T) (*Context, *Context) {
				httpCtx := Background().(*Context)
				return httpCtx, httpCtx
			},
		},
		{
			name: "copies of one request",
			setup: func(t *testing.T) (*Context, *Context) {
				base := Background().(*Context)
				writer := NewContext(base.Instance())
				reader := NewContext(base.Instance())

				require.Same(t, writer.values, reader.values)

				writer.WithValue("shared", "value")
				require.Equal(t, "value", reader.Value("shared"))

				return writer, reader
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, reader := tt.setup(t)

			runConcurrentContext(t, writer, reader)
		})
	}
}

// runConcurrentContext mirrors what database/sql does: one goroutine writes
// values through WithValue while another reads through Context, Done, Err and
// Value, exactly as database/sql.(*Rows).awaitDone does.
func runConcurrentContext(t *testing.T, writer, reader *Context) {
	t.Helper()

	const (
		goroutines = 10
		iterations = 1000
	)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			<-start
			for j := 0; j < iterations; j++ {
				writer.WithValue(idx, "value")
			}
		}(i)
	}

	// Key 0 is written above, so the reader asserts on a key that exists.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			<-start
			for j := 0; j < iterations; j++ {
				_ = reader.Context()
				_ = reader.Done()
				_ = reader.Err()
				_, _ = reader.Deadline()
				_ = reader.Value(0)
			}
		}()
	}

	close(start)
	wg.Wait()

	require.Equal(t, "value", reader.Value(0))
}
