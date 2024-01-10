package diagnostics

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	callTimeout    = 3 * time.Second
	testNetworkKey = "perf.network-health"
)

// Modules is all the registered modules on zbus
var Modules = []string{
	"storage",
	"node",
	"identityd",
	"vmd",
	"flist",
	"network",
	"container",
	"provision",
	"gateway",
	"qsfsd",
}

type moduleStatus struct {
	Status zbus.Status `json:"status,omitempty"`
	Err    error       `json:"error,omitempty"`
}

// Diagnostics show the health of zbus modules
type Diagnostics struct {
	// SystemStatusOk is the overall system status
	SystemStatusOk bool `json:"system_status_ok"`
	// ZosModules is a list of modules with their objects and workers
	ZosModules map[string]moduleStatus `json:"modules"`
	// Online is the state of the grid services reachable from the node
	Online bool `json:"online"`
}

func GetSystemDiagnostics(ctx context.Context, busClient zbus.Client, msgBrokerCon string) (Diagnostics, error) {
	results := Diagnostics{
		SystemStatusOk: true,
		ZosModules:     make(map[string]moduleStatus),
	}

	var wg sync.WaitGroup
	for _, module := range Modules {

		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			report := getModuleStatus(ctx, busClient, module)
			results.ZosModules[module] = report

			if report.Err != nil {
				results.SystemStatusOk = false
			}
		}(module)

	}

	wg.Wait()

	results.Online = isOnline(ctx, msgBrokerCon)

	return results, nil

}

func getModuleStatus(ctx context.Context, busClient zbus.Client, module string) moduleStatus {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	status, err := busClient.Status(ctx, module)
	return moduleStatus{
		Status: status,
		Err:    err,
	}
}

func isOnline(ctx context.Context, msgBrokerCon string) bool {
	conn, err := utils.NewRedisConn(msgBrokerCon)
	if err != nil {
		return false
	}
	defer conn.Close()

	data, err := conn.Do("GET", testNetworkKey)
	if err != nil {
		return false
	}

	var result perf.TaskResult
	if err := json.Unmarshal(data.([]byte), &result); err != nil {
		return false
	}

	for _, service := range result.Result.([]interface{}) {
		if !service.(map[string]interface{})["is_reachable"].(bool) {
			return false
		}
	}

	return true
}
