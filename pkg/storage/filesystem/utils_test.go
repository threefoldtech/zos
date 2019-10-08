package filesystem

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type TestExecuter struct {
	mock.Mock
}

func (t *TestExecuter) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	inputs := []interface{}{ctx, name}
	for _, arg := range args {
		inputs = append(inputs, arg)
	}

	result := t.Called(inputs...)
	return result.Get(0).([]byte), result.Error(1)
}
