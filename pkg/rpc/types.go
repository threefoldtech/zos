// AUTO-GENERATED: this file is auto generated please don't edit
package rpc

import "encoding/json"

type ZosRpcApi interface {
	SystemVersion(any, *Version) error
	SystemHypervisor(any, *string) error
	SystemDiagnostics(any, *Diagnostics) error
	SystemDmi(any, *DMI) error
	GpuList(any, *GPUs) error
	StorageMetrics(any, *PoolMetricsResult) error
	Statistics(any, *Counters) error
	PerfGetCpuBench(any, *CpuBenchTaskResult) error
	PerfGetHealth(any, *HealthTaskResult) error
	PerfGetIperf(any, *IperfTaskResult) error
	PerfGetPublicIP(any, *PublicIpTaskResult) error
	PerfGetAll(any, *AllTaskResult) error
	NetworkPrivateIps(string, *Ips) error
	NetworkPublicIps(any, *Ips) error
	NetworkHasIpv6(any, *bool) error
	NetworkInterfaces(any, *Interfaces) error
	NetworkPublicConfig(any, *PublicConfig) error
	NetworkWGPorts(any, *WGPorts) error
	AdminPublicNICSet(string, *any) error
	AdminPublicNICGet(any, *ExitDevice) error
	AdminInterfaces(any, *Interfaces) error
	DeploymentList(any, *Deployments) error
	DeploymentGet(uint64, *Deployment) error
	DeploymentChanges(uint64, *Workloads) error
	DeploymentUpdate(Deployment, *any) error
	DeploymentDeploy(Deployment, *any) error
}

type Interfaces struct {
	Interfaces []Interface `json:"interfaces"`
}

type CpuBenchTaskResult struct {
	Description string             `json:"description"`
	Timestamp   uint64             `json:"timestamp"`
	Result      CPUBenchmarkResult `json:"result"`
	Name        string             `json:"name"`
}

type DMI struct {
	Tooling  Tooling   `json:"tooling"`
	Sections []Section `json:"sections"`
}

type SubSection struct {
	Title      string         `json:"title"`
	Properties []PropertyData `json:"properties"`
}

type PropertyData struct {
	Val   string   `json:"value"`
	Items []string `json:"items"`
	Name  string   `json:"name"`
}

type ModuleStatus struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Err    string `json:"error"`
}

type Counters struct {
	Total  Capacity      `json:"total"`
	Used   Capacity      `json:"used"`
	System Capacity      `json:"system"`
	Users  UsersCounters `json:"users"`
}

type Capacity struct {
	SRU   uint64 `json:"sru"`
	HRU   uint64 `json:"hru"`
	MRU   uint64 `json:"mru"`
	IPV4U uint64 `json:"ipv4u"`
	CRU   uint64 `json:"cru"`
}

type Workload struct {
	Version     uint64          `json:"version"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	Metadata    string          `json:"metadata"`
	Description string          `json:"description"`
	Result      Result          `json:"result"`
}

type Deployment struct {
	Version              uint64               `json:"version"`
	TwinID               uint64               `json:"twin_id"`
	ContractID           uint64               `json:"contract_id"`
	Metadata             string               `json:"metadata"`
	Description          string               `json:"description"`
	Expiration           uint64               `json:"expiration"`
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	Workloads            []Workload           `json:"workloads"`
}

type Version struct {
	Zinit string `json:"zinit"`
	Zos   string `json:"zos"`
}

type Diagnostics struct {
	Healthy        bool           `json:"healthy"`
	SystemStatusOk bool           `json:"system_status_ok"`
	ZosModules     []ModuleStatus `json:"modules"`
}

type WorkerStatus struct {
	State     string `json:"state"`
	StartTime string `json:"time"`
	Action    string `json:"action"`
}

type PoolMetrics struct {
	Size uint64 `json:"size"`
	Used uint64 `json:"used"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type PublicIpTaskResult struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Timestamp   uint64     `json:"timestamp"`
	Result      []IPReport `json:"result"`
}

type CPUBenchmarkResult struct {
	Single    float64 `json:"single"`
	Multi     float64 `json:"multi"`
	Threads   uint64  `json:"threads"`
	Workloads uint64  `json:"workloads"`
}

type PublicConfig struct {
	Type   string `json:"type"`
	IPv4   string `json:"ipv4"`
	IPv6   string `json:"ipv6"`
	GW4    string `json:"gw4"`
	GW6    string `json:"gw6"`
	Domain string `json:"domain"`
}

type Deployments struct {
	Deployments []Deployment `json:"deployments"`
}

type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired uint64             `json:"weight_required"`
	Signatures     []Signature        `json:"signatures"`
	SignatureStyle string             `json:"signature_style"`
}

type SignatureRequest struct {
	Weight   uint64 `json:"weight"`
	TwinID   uint64 `json:"twin_id"`
	Required bool   `json:"required"`
}

type HealthTaskResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Timestamp   uint64         `json:"timestamp"`
	Result      []HealthReport `json:"result"`
}

type IPReport struct {
	Ip     string `json:"ip"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type UsersCounters struct {
	Deployments             uint64 `json:"deployments"`
	Workloads               uint64 `json:"workloads"`
	LastDeploymentTimestamp uint64 `json:"last_deployment_timestamp"`
}

type GPUs struct {
	GPUs []GPU `json:"gpus"`
}

type Workloads struct {
	Workloads []Workload `json:"workloads"`
}

type Result struct {
	Created uint64          `json:"created"`
	State   string          `json:"state"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

type CPUUtilizationPercent struct {
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
	HostTotal    float64 `json:"host_total"`
}

type Tooling struct {
	Aggregator string `json:"aggregator"`
	Decoder    string `json:"decoder"`
}

type Status struct {
	Objects []ObjectID     `json:"objects"`
	Workers []WorkerStatus `json:"workers"`
}

type HealthReport struct {
	Errors   []string `json:"errors"`
	TestName string   `json:"test_name"`
}

type IperfResult struct {
	NodeID        uint64                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
	UploadSpeed   float64               `json:"upload_speed"`
	DownloadSpeed float64               `json:"download_speed"`
}

type Ips struct {
	Ips []string `json:"ips"`
}

type Signature struct {
	TwinID        uint64 `json:"twin_id"`
	Signature     string `json:"signature"`
	SignatureType string `json:"signature_type"`
}

type ExitDevice struct {
	IsSingle        bool   `json:"is_single"`
	IsDual          bool   `json:"is_dual"`
	AsDualInterface string `json:"dual_interface"`
}

type IperfTaskResult struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Timestamp   uint64        `json:"timestamp"`
	Result      []IperfResult `json:"result"`
}

type ObjectID struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PoolMetricsResult struct {
	Pools []PoolMetrics `json:"pools"`
}

type GPU struct {
	ID       string `json:"id"`
	Vendor   string `json:"vendor"`
	Device   string `json:"device"`
	Contract uint64 `json:"contract"`
}

type AllTaskResult struct {
	HealthCheck  HealthTaskResult   `json:"health_check"`
	Iperf        IperfTaskResult    `json:"iperf"`
	PublicIp     PublicIpTaskResult `json:"public_ip"`
	CpuBenchmark CpuBenchTaskResult `json:"cpu_benchmark"`
}

type Section struct {
	SubSections []SubSection `json:"subsections"`
	HandleLine  string       `json:"handleline"`
	TypeStr     string       `json:"typestr"`
	Type        uint64       `json:"typenum"`
}

type WGPorts struct {
	Ports []uint64 `json:"ports"`
}

type Interface struct {
	Mac  string   `json:"mac,omitempty"`
	Name string   `json:"name"`
	Ips  []string `json:"ips"`
}
