package apigateway

import (
	"context"
)

func (g *apiGateway) statisticsGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.statisticsStub.GetCounters(ctx)
}
