package builders

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// ContainerBuilder is a struct that can build containers
type ContainerBuilder struct {
	workloads.Container
}

// NewContainerBuilder creates a new container builder
func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{}
}

// LoadContainerBuilder loads a container builder based on a file path
func LoadContainerBuilder(reader io.Reader) (*ContainerBuilder, error) {
	container := workloads.Container{}

	err := json.NewDecoder(reader).Decode(&container)
	if err != nil {
		return &ContainerBuilder{}, err
	}

	return &ContainerBuilder{Container: container}, nil
}

// Save saves the container builder to an IO.Writer
func (c *ContainerBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(c.Container)
	if err != nil {
		return err
	}
	return err
}

// Build validates and encrypts the secret environment of the container
func (c *ContainerBuilder) Build() error {
	// TODO check validity fields

	if c.Container.SecretEnvironment == nil {
		c.Container.SecretEnvironment = make(map[string]string)
	}

	for k, value := range c.Container.Environment {
		secret, err := encryptSecret(value, c.Container.NodeId)
		if err != nil {
			return errors.Wrapf(err, "failed to encrypt env with key '%s'", k)
		}
		c.Container.SecretEnvironment[k] = secret
	}
	c.Container.Environment = make(map[string]string)
	return nil
}

// WithNodeID sets the node ID to the container
func (c *ContainerBuilder) WithNodeID(nodeID string) {
	c.Container.NodeId = nodeID
}

// WithFlist sets the flist to the container
func (c *ContainerBuilder) WithFlist(flist string) {
	c.Container.Flist = flist
}

// WithHubURL sets the hub url to the container
func (c *ContainerBuilder) WithHubURL(url string) {
	c.Container.HubUrl = url
}

// WithEnvs sets the environments to the container
func (c *ContainerBuilder) WithEnvs(envs map[string]string) {
	c.Container.Environment = envs
}

// WithSecretEnvs sets the secret environments to the container
func (c *ContainerBuilder) WithSecretEnvs(envs map[string]string) {
	c.Container.SecretEnvironment = envs
}

// WithEntrypoint sets the entrypoint to the container
func (c *ContainerBuilder) WithEntrypoint(entrypoint string) {
	c.Container.Entrypoint = entrypoint
}

// WithVolumes sets the volumes to the container
func (c *ContainerBuilder) WithVolumes(mounts []workloads.ContainerMount) {
	c.Container.Volumes = mounts
}

// WithConnection sets the conntections to the container
func (c *ContainerBuilder) WithConnection(connections []workloads.NetworkConnection) {
	c.Container.NetworkConnection = connections
}

// WithStatsAggregator sets the stats aggregators to the container
func (c *ContainerBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) {
	c.Container.StatsAggregator = aggregators
}

// WithLogs sets the logs to the container
func (c *ContainerBuilder) WithLogs(logs []workloads.Logs) {
	c.Container.Logs = logs
}

// WithContainerCapacity sets the container capacity to the container
func (c *ContainerBuilder) WithContainerCapacity(cap workloads.ContainerCapacity) {
	c.Container.Capacity = cap
}
