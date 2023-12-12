package networkhealth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-redis/redis"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/perf"
)

type NetworkHealthTask struct{}

type ServiceStatus struct {
	Name        string
	IsReachable bool
	Err         error `json:"err,omitempty"`
}

var _ perf.Task = (*NetworkHealthTask)(nil)

func NewTask() *NetworkHealthTask {
	return &NetworkHealthTask{}
}

func (t *NetworkHealthTask) ID() string {
	return "network-health"
}

func (t *NetworkHealthTask) Description() string {
	return "Network health check runs periodically to check the connection to most of grid services."
}

func (t *NetworkHealthTask) Cron() string {
	return "0 */5 * * * *"
}

func (t *NetworkHealthTask) Jitter() uint32 {
	return 0
}

func (t *NetworkHealthTask) Run(ctx context.Context) (interface{}, error) {
	var err error
	result := []ServiceStatus{}

	env := environment.MustGet()

	status := pingHttp("activation-service", env.ActivationURL)
	result = append(result, status)

	status = pingHttp("graphql-gateway", env.GraphQL)
	result = append(result, status)

	status = pingRedis("hub", env.FlistURL)
	result = append(result, status)

	return result, err
}

func pingHttp(name, url string) ServiceStatus {
	report := ServiceStatus{
		Name:        name,
		IsReachable: true,
	}

	res, err := http.Get(url)
	if err != nil || res.StatusCode != http.StatusOK {
		report.IsReachable = false
		report.Err = err
	}
	defer res.Body.Close()

	return report
}

func pingRedis(name, url string) ServiceStatus {
	report := ServiceStatus{
		Name:        name,
		IsReachable: true,
	}

	addressSlice := strings.Split(url, "://")
	addr := ""
	if len(addr) == 1 {
		addr = addressSlice[0]
	} else {
		addr = addressSlice[1]
	}

	redis := redis.NewClient(&redis.Options{Addr: addr})
	res := redis.Ping()
	if res.Err() != nil {
		report.IsReachable = false
		report.Err = res.Err()
	}

	return report
}
