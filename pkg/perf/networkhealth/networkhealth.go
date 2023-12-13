package networkhealth

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/perf"
)

const requestTimeout = 5 * time.Second

type NetworkHealthTask struct{}

type ServiceStatus struct {
	Url         string `json:"url"`
	IsReachable bool   `json:"is_reachable"`
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
	env := environment.MustGet()
	servicesUrl := []string{
		env.ActivationURL, env.GraphQL, env.FlistURL,
	}
	servicesUrl = append(servicesUrl, env.SubstrateURL...)
	servicesUrl = append(servicesUrl, env.RelayURL...)

	reports := []ServiceStatus{}

	var wg sync.WaitGroup
	for _, serviceUrl := range servicesUrl {
		wg.Add(1)
		go func(serviceUrl string) {
			defer wg.Done()
			report := getNetworkReport(ctx, serviceUrl)
			reports = append(reports, report)
		}(serviceUrl)
	}
	wg.Wait()

	return reports, nil
}

func getNetworkReport(ctx context.Context, serviceUrl string) ServiceStatus {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	report := ServiceStatus{
		Url:         serviceUrl,
		IsReachable: true,
	}

	parsedUrl := parseUrl(serviceUrl)
	err := isReachable(ctx, parsedUrl)
	if err != nil {
		report.IsReachable = false
	}

	return report
}

func parseUrl(serviceUrl string) string {
	u, err := url.Parse(serviceUrl)
	if err != nil {
		return ""
	}
	host := u.Host

	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:80", host)
	}

	return host
}

func isReachable(ctx context.Context, serviceUrl string) error {
	d := net.Dialer{Timeout: requestTimeout}
	conn, err := d.DialContext(ctx, "tcp", serviceUrl)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	return nil
}
