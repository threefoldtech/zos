package apigateway

import (
	"context"
)

func (g *apiGateway) storagePoolsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.storageStub.Metrics(ctx)
}
