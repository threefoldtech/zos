package main

import (
	"time"
)

type ZosRpcApi interface {
	SystemVersion() (Version, error)
	SystemDmi() (DMI, error)
	SystemHypervior() (string, error)
	SystemDiagnostics() (Diagnostics, error)

	StorageMetrics() ([]PoolMetrics, error)

	Statistics() (Counters, error)

	PerfGet() (TaskResult, error)
	PerfGetAll() ([]TaskResult, error)
}

type Version struct {
	Zinit string
	Zos   string
}

type DMI struct {
	Tooling  Tooling
	Sections []Section
}

type Tooling struct {
	Aggregator string
	Decoder    string
}

type PropertyData struct {
	Val   string
	Items []string
}

type Section struct {
	HandleLine  string
	TypeStr     string
	Type        int
	SubSections []SubSection
}

type SubSection struct {
	Title      string
	Properties map[string]PropertyData
}

type ModuleStatus struct {
	Status Status
	Err    error
}

type Diagnostics struct {
	SystemStatusOk bool
	ZosModules     map[string]ModuleStatus
	Healthy        bool
}

type ObjectID struct {
	Name    string
	Version Version
}

type WorkerStatus struct {
	State     string
	StartTime time.Time
	Action    string
}

type Status struct {
	Objects []ObjectID
	Workers []WorkerStatus
}

type PoolMetrics struct {
	Name string
	Type string
	Size uint64
	Used uint64
}

type Counters struct {
	Total  Capacity
	Used   Capacity
	System Capacity
	Users  UsersCounters
}

type Capacity struct {
	CRU   uint64
	SRU   uint64
	HRU   uint64
	MRU   uint64
	IPV4U uint64
}

type UsersCounters struct {
	Deployments             int
	Workloads               int
	LastDeploymentTimestamp int64
}

type TaskResult struct {
	Name        string
	Description string
	Timestamp   uint64
	Result      interface{}
}
