package tpm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPersistedHandlers(t *testing.T) {
	t.Skip("manual test needs specific setup")
	handlers, err := PersistedHandlers(context.Background())
	assert.NoError(t, err)
	fmt.Println(handlers)
}

func TestPCRs(t *testing.T) {
	//t.Skip("manual test needs specific setup")
	pcrs, err := PCRs(context.Background())
	assert.NoError(t, err)
	fmt.Println(pcrs)
}
