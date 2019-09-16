package environment

import (
	"github.com/threefoldtech/zosv2/modules/kernel"
)

type EnvironmentManager struct {
	runningMode string

	bcdbUrl       string
	bcdbNamespace string
	bcdbPassword  string

	tnodbUrl string

	provisionTimeout  int64
	provisionInterval int64
}

const (
	runningDev  = "development"
	runningTest = "testing"
	runningMain = "production"
)

func GetEnvironment() EnvironmentManager {
	params := kernel.GetParams()
	return getEnvironmentFromParams(params)
}

func getEnvironmentFromParams(params kernel.Params) EnvironmentManager {
	var runmode []string

	runmode, found := params.Get("runmode")
	if !found {
		// fallback to default development
		runmode = make([]string, 1)
		runmode[0] = runningDev
	}

	switch runmode[0] {
	case runningTest:
		return EnvironmentManager{
			runningMode:       runningTest,
			bcdbUrl:           "10.10.10.0:8901",
			provisionTimeout:  120,
			provisionInterval: 120,
		}

	case runningMain:
		return EnvironmentManager{
			runningMode:       runningMain,
			bcdbUrl:           "1.2.3.4:8901",
			provisionTimeout:  240,
			provisionInterval: 240,
		}

	case runningDev:
		fallthrough

	default:
		return EnvironmentManager{
			runningMode:       runningDev,
			tnodbUrl:          "https://tnodb.dev.grid.tf",
			provisionTimeout:  60,
			provisionInterval: 60,
		}
	}
}
