package environment

import (
	"github.com/threefoldtech/zosv2/modules/kernel"
	"os"
	"strconv"
)

type EnvironmentManager struct {
	RunningMode string

	BcdbUrl       string
	BcdbNamespace string
	BcdbPassword  string

	TnodbUrl string

	ProvisionTimeout  int64
	ProvisionInterval int64
}

const (
	RunningDev  = "development"
	RunningTest = "testing"
	RunningMain = "production"
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
		runmode[0] = RunningDev
	}

	switch runmode[0] {
	case RunningTest:
		env = EnvironmentManager{
			RunningMode:       RunningTest,
			BcdbUrl:           "10.10.10.0:8901",
			ProvisionTimeout:  120,
			ProvisionInterval: 120,
		}

	case RunningMain:
		env = EnvironmentManager{
			RunningMode:       RunningMain,
			BcdbUrl:           "1.2.3.4:8901",
			ProvisionTimeout:  240,
			ProvisionInterval: 240,
		}

	case RunningDev:
		fallthrough

	default:
		env = EnvironmentManager{
			RunningMode:       RunningDev,
			TnodbUrl:          "https://tnodb.dev.grid.tf",
			ProvisionTimeout:  60,
			ProvisionInterval: 60,
		}
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_BCDB_URL"); e != "" {
		env.BcdbUrl = e
	}

	if e := os.Getenv("ZOS_BCDB_NAMESPACE"); e != "" {
		env.BcdbNamespace = e
	}

	if e := os.Getenv("ZOS_BCDB_PASSWORD"); e != "" {
		env.BcdbPassword = e
	}

	if e := os.Getenv("ZOS_TNODB_URL"); e != "" {
		env.TnodbUrl = e
	}

	if e := os.Getenv("ZOS_PROVISION_INTERVAL"); e != "" {
		if i, err := strconv.ParseInt(e, 10, 64); err == nil {
			env.ProvisionInterval = i
		}
	}

	if e := os.Getenv("ZOS_PROVISION_TIMEOUT"); e != "" {
		if i, err := strconv.ParseInt(e, 10, 64); err == nil {
			env.ProvisionTimeout = i
		}
	}

	return env
}
