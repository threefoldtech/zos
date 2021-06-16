package rmb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	systemLocalBus = "msgbus.system.local"
	replyBus       = "msgbus.system.reply"
	numWorkers     = 5
)

// TwinKeyID is where the twin key is stored
type TwinKeyID struct{}

// MessageKey is where the original message is stored
type MessageKey struct{}

// Message is an struct used to communicate over the messagebus
type Message struct {
	Version    int      `json:"ver"`
	UID        string   `json:"uid"`
	Command    string   `json:"cmd"`
	Expiration int      `json:"exp"`
	Retry      int      `json:"try"`
	Data       string   `json:"dat"`
	TwinSrc    uint32   `json:"src"`
	TwinDest   []uint32 `json:"dst"`
	Retqueue   string   `json:"ret"`
	Schema     string   `json:"shm"`
	Epoch      int64    `json:"now"`
	Err        string   `json:"err"`
}

type messageBusSubrouter struct {
	handlers map[string]Handler
	sub      map[string]*messageBusSubrouter
	mw       []Middleware
}

func newSubRouter() messageBusSubrouter {
	return messageBusSubrouter{
		handlers: make(map[string]Handler),
		sub:      make(map[string]*messageBusSubrouter),
	}
}

func (m *messageBusSubrouter) call(ctx context.Context, route string, payload []byte) (result interface{}, err error) {
	for _, mw := range m.mw {
		ctx, err = mw(ctx, payload)
		if err != nil {
			return nil, err
		}
	}

	handler, ok := m.handlers[route]
	if ok {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("handler panicked with: %s", err)
			}
		}()

		result, err = handler(ctx, payload)
		return
	}

	parts := strings.SplitN(route, ".", 2)

	key := parts[0]
	var subroute string
	if len(parts) == 2 {
		subroute = parts[1]
	}

	router, ok := m.sub[key]
	if !ok {
		return nil, ErrFunctionNotFound
	}

	return router.call(ctx, subroute, payload)
}

func (m *messageBusSubrouter) Use(mw Middleware) {
	m.mw = append(m.mw, mw)
}

func (m *messageBusSubrouter) Subroute(prefix string) Router {
	//r.handle('abc.def.fun', handler)
	// sub = r.handle(xyz)
	// sub.use(middle)
	// sub.handle('func', handler) // xyz.func
	if strings.Contains(prefix, ".") {
		panic("invalid subrouter prefix should not have '.'")
	}

	sub, ok := m.sub[prefix]
	if ok {
		return sub
	}

	r := newSubRouter()
	m.sub[prefix] = &r
	return &r
}

// WithHandler adds a topic handler to the messagebus
func (m *messageBusSubrouter) WithHandler(topic string, handler Handler) {
	if _, ok := m.handlers[topic]; ok {
		panic("handler already registered")
	}

	m.handlers[topic] = handler
}

func (m *messageBusSubrouter) getTopics(prefix string, l *[]string) {
	for r := range m.handlers {
		if len(prefix) != 0 {
			r = fmt.Sprintf("%s.%s", prefix, r)
		}
		*l = append(*l, r)
	}

	for r, sub := range m.sub {
		if len(prefix) != 0 {
			r = fmt.Sprintf("%s.%s", prefix, r)
		}
		sub.getTopics(r, l)
	}

}

// MessageBus is a struct that contains everything required to run the message bus
type MessageBus struct {
	messageBusSubrouter
	pool *redis.Pool
}

// New creates a new message bus
func New(address string) (*MessageBus, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", address)
	}

	return &MessageBus{
		pool:                pool,
		messageBusSubrouter: newSubRouter(),
	}, nil
}

func (m *MessageBus) Handlers() []string {
	topics := make([]string, 0)
	m.getTopics("", &topics)

	return topics
}

func (m *MessageBus) getOne(args redis.Args) ([][]byte, error) {
	con := m.pool.Get()
	defer con.Close()

	data, err := redis.ByteSlices(con.Do("BLPOP", args...))
	if err != nil && err != redis.ErrNil {
		return nil, err
	}

	if err == redis.ErrNil || data == nil {
		//timeout, just try again immediately
		return nil, redis.ErrNil
	}

	return data, nil
}

// Run runs listeners to the configured handlers
// and will trigger the handlers in the case an event comes in
func (m *MessageBus) Run(ctx context.Context) error {
	con := m.pool.Get()
	defer con.Close()

	topics := m.Handlers()
	for i, topic := range topics {
		topics[i] = "msgbus." + topic
	}

	jobs := make(chan Message, numWorkers)
	for i := 1; i <= numWorkers; i++ {
		go m.worker(ctx, jobs)
	}

	args := redis.Args{}.AddFlat(topics).Add(10)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		data, err := m.getOne(args)

		if err == redis.ErrNil {
			continue
		} else if err != nil {
			log.Err(err).Msg("failed to read from system local messagebus, retry in 2 seconds")
			<-time.After(2 * time.Second)
			continue
		}

		var message Message
		err = json.Unmarshal(data[1], &message)
		if err != nil {
			log.Err(err).Msg("failed to unmarshal message")
			continue
		}

		select {
		case jobs <- message:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *MessageBus) worker(ctx context.Context, jobs chan Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-jobs:
			bytes, err := message.GetPayload()
			if err != nil {
				log.Err(err).Msg("err while parsing payload reply")
			}

			requestCtx := context.WithValue(ctx, TwinKeyID{}, message.TwinSrc)
			requestCtx = context.WithValue(requestCtx, MessageKey{}, message)

			data, err := m.call(requestCtx, message.Command, bytes)
			if err != nil {
				log.Err(err).Msg("err while handling job")
				// TODO: create an error object
				message.Err = err.Error()
			}

			err = m.sendReply(message, data)
			if err != nil {
				log.Err(err).Msg("err while sending reply")
			}
		}
	}
}

// GetMessage gets a message from the context, panics if it's not there
func GetMessage(ctx context.Context) (*Message, error) {
	message, ok := ctx.Value(MessageKey{}).(Message)
	if !ok {
		panic("failed to load message from context")
	}

	return &message, nil
}

// sendReply send a reply to the message bus with some data
func (m *MessageBus) sendReply(message Message, data interface{}) error {
	con := m.pool.Get()
	defer con.Close()

	src := message.TwinDest[0]
	// reply to source
	message.TwinDest = []uint32{message.TwinSrc}
	message.TwinSrc = src

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// base 64 encode the response data
	message.Data = base64.StdEncoding.EncodeToString(bytes)

	// set the time to now
	message.Epoch = time.Now().Unix()

	bytes, err = json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = con.Do("LPUSH", message.Retqueue, string(bytes))
	if err != nil {
		log.Err(err).Msg("failed to push to reply messagebus")
		return err
	}

	return nil
}

// PushMessage pushes a message to a topic
// for testing purposes
func (m *MessageBus) PushMessage(topic string, message Message) error {
	con := m.pool.Get()
	defer con.Close()

	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = con.Do("RPUSH", topic, bytes)
	if err != nil {
		log.Err(err).Msg("failed to push to topic")
		return err
	}

	return nil
}

// GetPayload returns the payload for a message's data
func (m *Message) GetPayload() ([]byte, error) {
	return base64.StdEncoding.DecodeString(m.Data)
}

func newRedisPool(address string) (*redis.Pool, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "tcp":
		host = u.Host
	case "unix":
		host = u.Path
	default:
		return nil, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}
	var opts []redis.DialOption

	if u.User != nil {
		opts = append(
			opts,
			redis.DialPassword(u.User.Username()),
		)
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(u.Scheme, host, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   5,
		MaxIdle:     3,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}
