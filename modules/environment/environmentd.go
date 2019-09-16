package environment

import (
	"github.com/threefoldtech/zosv2/modules/kernel"
	"os"
	"strconv"
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
	var env EnvironmentManager

	runmode, found := params.Get("runmode")
	if !found {
		// Fallback to default development mode
		runmode = make([]string, 1)
		runmode[0] = runningDev
	}

	switch runmode[0] {
	case runningTest:
		env = EnvironmentManager{
			runningMode:       runningTest,
			bcdbUrl:           "10.10.10.0:8901",
			provisionTimeout:  120,
			provisionInterval: 120,
		}

	case runningMain:
		env = EnvironmentManager{
			runningMode:       runningMain,
			bcdbUrl:           "1.2.3.4:8901",
			provisionTimeout:  240,
			provisionInterval: 240,
		}

	case runningDev:
		fallthrough

	default:
		env = EnvironmentManager{
			runningMode:       runningDev,
			tnodbUrl:          "https://tnodb.dev.grid.tf",
			provisionTimeout:  60,
			provisionInterval: 60,
		}
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_BCDB_URL"); e != "" {
		env.bcdbUrl = e
	}

	if e := os.Getenv("ZOS_BCDB_NAMESPACE"); e != "" {
		env.bcdbNamespace = e
	}

	if e := os.Getenv("ZOS_BCDB_PASSWORD"); e != "" {
		env.bcdbPassword = e
	}

	if e := os.Getenv("ZOS_TNODB_URL"); e != "" {
		env.tnodbUrl = e
	}

	if e := os.Getenv("ZOS_PROVISION_INTERVAL"); e != "" {
		if i, err := strconv.ParseInt(e, 10, 64); err == nil {
			env.provisionInterval = i
		}
	}

	if e := os.Getenv("ZOS_PROVISION_TIMEOUT"); e != "" {
		if i, err := strconv.ParseInt(e, 10, 64); err == nil {
			env.provisionTimeout = i
		}
	}

	return env
}
