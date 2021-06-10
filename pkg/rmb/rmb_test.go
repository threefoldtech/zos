package rmb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	require := require.New(t)

	router := newSubRouter()

	require.NotNil(router)
	router.WithHandler("test.handler.do", func(ctx context.Context, payload []byte) (interface{}, error) {
		return nil, nil
	})

	sub := router.Subroute("test2")

	sub.WithHandler("handler.do", func(ctx context.Context, payload []byte) (interface{}, error) {
		return nil, nil
	})

	l := make([]string, 0)
	router.getTopics("", &l)

	require.Len(l, 2)
	require.Equal([]string{"test.handler.do", "test2.handler.do"}, l)
}
