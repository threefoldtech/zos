package gateway

import (
	"bufio"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/namespace"
)

const (
	publicNS   = "public"
	metricsURL = "http://172.0.0.1:8082"

	metricReceived = "traefik_service_bytes_received_total"
	metricSent     = "traefik_service_bytes_sent_total"
)

var (
	ErrMetricsNotAvailable = errors.New("errors not available")

	metricM = regexp.MustCompile(`^(\w+)({[^}]+})? ([0-9e\+-]+)`)
	tagsM   = regexp.MustCompile(`([^,={]+)="([^"]+)"`)
)

type value struct {
	value  float64
	labels map[string]string
}

type metric struct {
	key    string
	values []value
}

// group group values by label, if multiple values in this
// metric has the same value for the label, values are added.
func (m *metric) group(l string) map[string]float64 {
	result := make(map[string]float64)
	for _, v := range m.values {
		result[v.labels[l]] += v.value
	}
	return result
}

func tags(s string) map[string]string {
	matches := tagsM.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, m := range matches {
		result[m[1]] = m[2]
	}
	return result
}

func parseMetrics(in io.Reader) (map[string]*metric, error) {
	scanner := bufio.NewScanner(in)
	results := make(map[string]*metric)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := metricM.FindStringSubmatch(line)

		if len(parts) == 0 {
			// no match
			continue
		}
		key := parts[1]
		labels := tags(parts[2])
		valueStr := parts[3]

		valueF, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse value '%s' for metric '%s'", valueStr, key)
		}

		v := value{
			value:  valueF,
			labels: labels,
		}

		m, ok := results[key]
		if !ok {
			m = &metric{key: key}
			results[key] = m
		}

		m.values = append(m.values, v)
	}

	return results, nil
}
func metrics(url string) (map[string]*metric, error) {
	//"http://localhost:9090/metrics"
	response, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics")
	}

	defer func() {
		_, _ = io.ReadAll(response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode != 200 {
		return nil, errors.Wrapf(err, "got wrong status code for metrics: %s", response.Status)
	}

	return parseMetrics(response.Body)
}

func (g *gatewayModule) Metrics() (result pkg.GatewayMetrics, err error) {
	// metric is only available if traefik is running. we can instead of doing
	// all the checks, we can try to directly get the metrics and see if we
	// can get it. we need to do these operations anyway.
	pubNS, err := namespace.GetByName(publicNS)
	if errors.Is(err, ns.NSPathNotExistErr{}) {
		return result, ErrMetricsNotAvailable
	} else if err != nil {
		return result, errors.Wrap(err, "failed to get public namespace")
	}

	defer pubNS.Close()
	var values map[string]*metric
	err = pubNS.Do(func(_ ns.NetNS) error {
		values, err = metrics(metricsURL)
		if err != nil {
			return ErrMetricsNotAvailable
		}

		return nil
	})

	if err != nil {
		return result, err
	}

	if m, ok := values[metricSent]; ok {
		// sent metrics.
		result.Sent = m.group("service")
	}

	if m, ok := values[metricReceived]; ok {
		result.Received = m.group("service")
	}

	return
}
