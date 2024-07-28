package pkg

//go:generate zbusc -module node -version 0.0.1 -name performance-monitor -package stubs github.com/threefoldtech/zos/pkg+PerformanceMonitor stubs/performance_monitor_stub.go

var (
	HealthCheckTaskName  = "healthcheck"
	IperfTaskName        = "iperf"
	PublicIpTaskName     = "public-ip-validation"
	CpuBenchmarkTaskName = "cpu-benchmark"
)

type PerformanceMonitor interface {
	GetAllTaskResult() (AllTaskResult, error)
	GetIperfTaskResult() (IperfTaskResult, error)
	GetHealthTaskResult() (HealthTaskResult, error)
	GetPublicIpTaskResult() (PublicIpTaskResult, error)
	GetCpuBenchTaskResult() (CpuBenchTaskResult, error)
	// Deprecated
	Get(taskName string) (TaskResult, error)
	GetAll() ([]TaskResult, error)
}

type CPUBenchmarkResult struct {
	SingleThreaded float64 `json:"single"`
	MultiThreaded  float64 `json:"multi"`
	Threads        int     `json:"threads"`
	Workloads      int     `json:"workloads"`
}

type CpuBenchTaskResult struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Timestamp   uint64             `json:"timestamp"`
	Result      CPUBenchmarkResult `json:"result"`
}

type AllTaskResult struct {
	CpuBenchmark CpuBenchTaskResult `json:"cpu_benchmark"`
	HealthCheck  HealthTaskResult   `json:"health_check"`
	Iperf        IperfTaskResult    `json:"iperf"`
	PublicIp     PublicIpTaskResult `json:"public_ip"`
}

type HealthReport struct {
	TestName string   `json:"test_name"`
	Errors   []string `json:"errors"`
}

type HealthTaskResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Timestamp   uint64         `json:"timestamp"`
	Result      []HealthReport `json:"result"`
}

type Report struct {
	Ip     string `json:"ip"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type PublicIpTaskResult struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Timestamp   uint64   `json:"timestamp"`
	Result      []Report `json:"result"`
}

// TaskResult the result test schema
type TaskResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Timestamp   uint64      `json:"timestamp"`
	Result      interface{} `json:"result"`
}

type IperfTaskResult struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Timestamp   uint64        `json:"timestamp"`
	Result      []IperfResult `json:"result"`
}

// IperfResult for iperf test results
type IperfResult struct {
	UploadSpeed   float64               `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64               `json:"download_speed"` // in bit/sec
	NodeID        uint32                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
}

type CPUUtilizationPercent struct {
	HostTotal    float64 `json:"host_total"`
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
}
