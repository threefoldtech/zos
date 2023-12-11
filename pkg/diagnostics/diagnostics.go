package diagnostics

import (
	"context"
	"sync"
	"time"

	"github.com/threefoldtech/zbus"
)

const callTimeout = 3 * time.Second

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
	// Modules is a list of modules with their objects and workers
	Modules map[string]moduleStatus `json:"modules"`
}

func GetSystemDiagnostics(ctx context.Context, busClient zbus.Client) (Diagnostics, error) {
	results := Diagnostics{
		SystemStatusOk: true,
		Modules:        make(map[string]moduleStatus),
	}

	var wg sync.WaitGroup
	for _, module := range Modules {

		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			report := getModuleStatus(ctx, busClient, module)
			results.Modules[module] = report

			if report.Err != nil {
				results.SystemStatusOk = false
			}
		}(module)

	}

	wg.Wait()

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
