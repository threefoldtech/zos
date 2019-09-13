package environment

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
)

type environmentManager struct {
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

func NewManager() (modules.EnvironmentManager, error) {
	fmt.Println("New Environment Manager")

	// TODO: detect running environment

	env := &environmentManager{
		runningMode: runningDev,
	}

	if env.runningMode == runningDev {
		env.tnodbUrl = "https://tnodb.dev.grid.tf"
		env.provisionTimeout = 60
		env.provisionInterval = 60
	}

	if env.runningMode == runningTest {
		env.bcdbUrl = "10.10.10.0:8901"
		env.provisionTimeout = 120
		env.provisionInterval = 120
	}

	if env.runningMode == runningMain {
		env.bcdbUrl = "1.2.3.4:8901"
		env.provisionTimeout = 240
		env.provisionInterval = 240
	}

	return env, nil
}

func (e *environmentManager) Something() error {
	return nil
}
