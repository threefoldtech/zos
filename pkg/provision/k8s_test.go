package provision

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestResult(t *testing.T) {
	result := K8sClusterResult{}
	b, err := ioutil.ReadFile("/home/zaibon/k3s.yaml")
	require.NoError(t, err)

	err = yaml.Unmarshal(b, &result)
	require.NoError(t, err)

	fmt.Printf("%+v", result)
}
