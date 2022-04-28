package vm

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	streamPrefix = "stream:"
)

// StreamCreate creates a stream for vm `name`
func (m *Module) StreamCreate(name string, stream pkg.Stream) error {
	if err := stream.Valid(); err != nil {
		return err
	}

	id := fmt.Sprintf("%s%s", streamPrefix, stream.ID)

	_, err := Find(name)
	if err != nil {
		return err
	}

	file := m.logsPath(name)

	cl := zinit.Default()
	if _, err := cl.Get(stream.ID); err == nil {
		return fmt.Errorf("stream with same id '%s' already exists", id)
	}

	cmd := fmt.Sprintf("tailstream -o %s %s", quote(stream.Output), file)
	if stream.Namespace != "" {
		cmd = fmt.Sprintf("ip netns exec %s %s", stream.Namespace, cmd)
	}

	service := zinit.InitService{
		Exec: cmd,
		Log:  zinit.NoneLogType,
	}

	if err := zinit.AddService(id, service); err != nil {
		return errors.Wrapf(err, "failed to add stream service '%s'", id)
	}

	return cl.Monitor(id)
}

// delete stream by stream id.
func (m *Module) StreamDelete(id string) error {
	id = fmt.Sprintf("%s%s", streamPrefix, id)
	cl := zinit.Default()

	defer func() {
		_ = zinit.RemoveService(id)
	}()

	_, err := cl.Get(id)
	if errors.Is(err, zinit.ErrUnknownService) {
		return nil
	}

	if err := cl.StopWait(30*time.Second, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to stop stream service")
	}

	return cl.Forget(id)
}
