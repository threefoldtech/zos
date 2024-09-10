package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/threefoldtech/zos/pkg/environment"
)

const defaultRolloutConfigURL = "https://raw.githubusercontent.com/threefoldtech/zos/development_rollout_update/pkg/upgrade/config.json"

// RolloutConfig is the configuration for A/B testing before zos update
type RolloutConfig struct {
	TestFarms     []uint32 `json:"test_farms"`
	SafeToUpgrade bool     `json:"safe_to_upgrade"`
}

func parseRolloutConfig(file io.Reader) (map[string]RolloutConfig, error) {
	var conf map[string]RolloutConfig

	configFile, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rollout config file: %+w", err)
	}

	if err = json.Unmarshal(configFile, &conf); err != nil {
		return nil, err
	}

	for network := range conf {
		if network != environment.RunningDev.String() && network != environment.RunningQA.String() && network != environment.RunningTest.String() && network != environment.RunningMain.String() {
			return nil, fmt.Errorf("invalid network passed: %s", network)
		}
	}

	return conf, nil
}

func readRolloutConfig(env string) (RolloutConfig, error) {
	response, err := http.Get(defaultRolloutConfigURL)
	if err != nil {
		return RolloutConfig{}, err
	}

	defer response.Body.Close()

	rolloutConfig, err := parseRolloutConfig(response.Body)
	if err != nil {
		return RolloutConfig{}, err
	}

	cfg, ok := rolloutConfig[env]
	if !ok {
		return RolloutConfig{}, fmt.Errorf("env '%s' does not exist in the configs", env)
	}

	return cfg, nil
}
