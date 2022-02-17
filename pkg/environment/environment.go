package environment

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/kernel"
)

const (
	// SubstrateDevURL default substrate url
	SubstrateDevURL  = "wss://tfchain.dev.grid.tf/"
	ActivationDevURL = "https://activation.dev.grid.tf/activation/activate"

	// SubstrateDevURL default substrate url
	SubstrateTestURL  = "wss://tfchain.test.grid.tf/"
	ActivationTestURL = "https://activation.test.grid.tf/activation/activate"

	SubstrateMainURL  = "wss://tfchain.grid.tf/"
	ActivationMainURL = "https://activation.grid.tf/activation/activate"

	BaseExtendedURL = "https://raw.githubusercontent.com/threefoldtech/zos-config/main/"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunningMode

	FlistURL string
	BinRepo  string

	FarmerID pkg.FarmID
	Orphan   bool

	FarmSecret    string
	SubstrateURL  string
	ActivationURL string

	ExtendedConfigURL string
}

// RunningMode type
type RunningMode string

func (r RunningMode) String() string {
	switch r {
	case RunningDev:
		return "development"
	case RunningMain:
		return "production"
	case RunningTest:
		return "testing"
	}

	return "unknown"
}

// Possible running mode of a node
const (
	//RunningDev   RunningMode = "dev"
	RunningDev RunningMode = "dev"
	//RunningTest  RunningMode = "test"
	RunningTest RunningMode = "test"
	RunningMain RunningMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  pkg.FarmID = 0
	OrphanageTest pkg.FarmID = 0
	OrphanageMain pkg.FarmID = 0
)

var (
	envDev = Environment{
		RunningMode:   RunningDev,
		SubstrateURL:  SubstrateDevURL,
		ActivationURL: ActivationDevURL,
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins.dev",
	}

	envTest = Environment{
		RunningMode: RunningTest,
		// TODO: this should become a different substrate ?
		SubstrateURL:  SubstrateTestURL,
		ActivationURL: ActivationTestURL,
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins.test",
	}

	// same as testnet for now. will be updated the day of the launch of production network
	envProd = Environment{
		RunningMode:   RunningMain,
		SubstrateURL:  SubstrateMainURL,
		ActivationURL: ActivationMainURL,
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins",
	}
)

// MustGet returns the running environment of the node
// panics on error
func MustGet() Environment {
	env, err := Get()
	if err != nil {
		panic(err)
	}

	return env
}

// Get return the running environment of the node
func Get() (Environment, error) {
	params := kernel.GetParams()
	return getEnvironmentFromParams(params)
}

// GetSubstrate gets a client to subsrate blockchain
func (e *Environment) GetSubstrate() (*substrate.Substrate, error) {
	return substrate.NewSubstrate(e.SubstrateURL)
}

func getEnvironmentFromParams(params kernel.Params) (Environment, error) {
	var env Environment
	runmode := ""
	if modes, ok := params.Get("runmode"); ok {
		if len(modes) >= 1 {
			runmode = modes[0]
		}
	} else {
		runmode = os.Getenv("ZOS_RUNMODE")
	}

	if len(runmode) == 0 {
		runmode = string(RunningMain)
	}

	switch RunningMode(runmode) {
	case RunningDev:
		env = envDev
	case RunningTest:
		env = envTest
	case RunningMain:
		env = envProd
	default:
		env = envProd
	}

	extended, found := params.Get("config_url")
	if found && len(extended) >= 1 {
		env.ExtendedConfigURL = extended[0]
	}

	if substrate, ok := params.Get("substrate"); ok {
		if len(substrate) > 0 {
			env.SubstrateURL = substrate[len(substrate)-1]
		}
	}

	if activation, ok := params.Get("activation"); ok {
		if len(activation) > 0 {
			env.ActivationURL = activation[len(activation)-1]
		}
	}

	if farmSecret, ok := params.Get("secret"); ok {
		if len(farmSecret) > 0 {
			env.FarmSecret = farmSecret[len(farmSecret)-1]
		}
	}

	farmerID, found := params.Get("farmer_id")
	if !found || len(farmerID) < 1 || farmerID[0] == "" {
		// fmt.Println("Warning: no valid farmer_id found in kernel parameter, fallback to orphanage")
		env.Orphan = true

		switch env.RunningMode {
		case RunningDev:
			env.FarmerID = OrphanageDev
		case RunningTest:
			env.FarmerID = OrphanageTest
		case RunningMain:
			env.FarmerID = OrphanageMain
		}

	} else {
		env.Orphan = false
		id, err := strconv.ParseUint(farmerID[0], 10, 32)
		if err != nil {
			return env, errors.Wrap(err, "wrong format for farm ID")
		}
		env.FarmerID = pkg.FarmID(id)
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_SUBSTRATE_URL"); e != "" {
		env.SubstrateURL = e
	}

	if e := os.Getenv("ZOS_FLIST_URL"); e != "" {
		env.FlistURL = e
	}

	if e := os.Getenv("ZOS_BIN_REPO"); e != "" {
		env.BinRepo = e
	}

	return env, nil
}
