package apigateway

import (
	"context"
)

func (g *apiGateway) gpuListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.statisticsStub.ListGPUs(ctx)
}
