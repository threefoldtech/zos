package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/rmb/direct/types"
)

type directClient struct {
	id        uint32
	signer    substrate.Identity
	con       *websocket.Conn
	responses map[string]chan *types.Envelope
	m         sync.Mutex
}

// id is the twin id that is associated with the given identity.
func NewClient(ctx context.Context, url string, id uint32, identity substrate.Identity) (rmb.Client, error) {
	token, err := NewJWT(id, identity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build authentication token")
	}
	url = fmt.Sprintf("%s?%s", url, token)

	con, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect")
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("invalid response %s", resp.Status)
	}

	cl := &directClient{
		id:        id,
		signer:    identity,
		con:       con,
		responses: make(map[string]chan *types.Envelope),
	}

	go cl.process()
	return cl, nil
}

func (d *directClient) process() {
	defer d.con.Close()
	// todo: set error on connection here
	for {
		typ, msg, err := d.con.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("websocket error connection closed")
			return
		}
		if typ != websocket.BinaryMessage {
			continue
		}

		var env types.Envelope
		if err := proto.Unmarshal(msg, &env); err != nil {
			log.Error().Err(err).Msg("invalid message payload")
			return
		}

		d.router(&env)
	}
}

func (d *directClient) router(env *types.Envelope) {
	d.m.Lock()
	defer d.m.Unlock()

	ch, ok := d.responses[env.Uid]
	if !ok {
		return
	}

	select {
	case ch <- env:
	default:
		// client is not waiting anymore! just return then
	}
}

func (d *directClient) makeRequest(dest uint32, cmd string, data []byte, ttl uint64) (*types.Envelope, error) {
	env := types.Envelope{
		Uid:         uuid.NewString(),
		Timestamp:   uint64(time.Now().Unix()),
		Expiration:  ttl,
		Source:      d.id,
		Destination: dest,
	}

	env.Message = &types.Envelope_Request{
		Request: &types.Request{
			Command: cmd,
			Data:    data,
		},
	}

	toSign, err := Challenge(&env)
	if err != nil {
		return nil, err
	}

	env.Signature, err = Sign(d.signer, toSign)
	if err != nil {
		return nil, err
	}

	return &env, nil

}

func (d *directClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	var ttl uint64 = 5 * 60
	deadline, ok := ctx.Deadline()
	if ok {
		ttl = uint64(time.Until(deadline).Seconds())
	}

	request, err := d.makeRequest(twin, fn, payload, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	ch := make(chan *types.Envelope)
	d.m.Lock()
	d.responses[request.Uid] = ch
	d.m.Unlock()

	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	if err := d.con.WriteMessage(websocket.BinaryMessage, bytes); err != nil {
		return err
	}

	var response *types.Envelope
	select {
	case <-ctx.Done():
		return ctx.Err()
	case response = <-ch:
	}
	if response == nil {
		// shouldn't happen but just in case
		return fmt.Errorf("no response received")
	}

	//TODO: signature verification must be done here

	resp := response.GetResponse()
	if resp == nil {
		return fmt.Errorf("received a non response envelope")
	}

	errResp := resp.GetError()
	if errResp != nil {
		// include code also
		return fmt.Errorf(errResp.Message)
	}

	if result == nil {
		return nil
	}

	reply := resp.GetReply()

	return json.Unmarshal(reply.Data, &result)
}
