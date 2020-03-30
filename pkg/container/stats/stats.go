package stats

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	v1 "github.com/containerd/cgroups/stats/v1"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/typeurl"
	"github.com/rs/zerolog/log"
)

// Logs defines a custom backend with variable settings
type StatsSet struct {
	MemoryUsage uint64 `json:"memory_usage"`
	MemoryLimit uint64 `json:"memory_limit"`
	MemoryCache uint64 `json:"memory_cache"`
	CpuUsage    uint32 `json:"cpu_usage"`
	PidsCurrent uint64 `json:"pids_current"`
}

func Monitor(addr string, ns string, task containerd.Task) error {
	client, err := containerd.New(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	for {
		monitor(ctx, task)
		time.Sleep(2 * time.Second)
	}
}

func monitor(ctx context.Context, task containerd.Task) error {
	metric, err := task.Metrics(ctx)
	if err != nil {
		log.Error().Err(err).Msg("metrics")
		return err
	}

	anydata, err := typeurl.UnmarshalAny(metric.Data)
	if err != nil {
		return err
	}

	var data *v1.Metrics
	switch v := anydata.(type) {
	case *v1.Metrics:
		data = v
	default:
		return fmt.Errorf("wrong metric type")
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 8, 4, ' ', 0)
	fmt.Fprintf(w, "ID\tTIMESTAMP\t\n")
	fmt.Fprintf(w, "%s\t%s\t\n\n", metric.ID, metric.Timestamp)

	printCgroupMetricsTable(w, data)
	w.Flush()

	return nil
}

func printCgroupMetricsTable(w *tabwriter.Writer, data *v1.Metrics) {
	fmt.Fprintf(w, "METRIC\tVALUE\t\n")
	if data.Memory != nil {
		fmt.Fprintf(w, "memory.usage_in_bytes\t%d\t\n", data.Memory.Usage.Usage)
		fmt.Fprintf(w, "memory.limit_in_bytes\t%d\t\n", data.Memory.Usage.Limit)
		fmt.Fprintf(w, "memory.stat.cache\t%d\t\n", data.Memory.TotalCache)
	}
	if data.CPU != nil {
		fmt.Fprintf(w, "cpuacct.usage\t%d\t\n", data.CPU.Usage.Total)
		fmt.Fprintf(w, "cpuacct.usage_percpu\t%v\t\n", data.CPU.Usage.PerCPU)
	}
	if data.Pids != nil {
		fmt.Fprintf(w, "pids.current\t%v\t\n", data.Pids.Current)
		fmt.Fprintf(w, "pids.limit\t%v\t\n", data.Pids.Limit)
	}
}
