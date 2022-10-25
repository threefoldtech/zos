package mw

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/provision"
)

type twinPublicKeyID struct{}
type twinKeyID struct{}

// TwinPublicKey extracts twin public key from request
func TwinPublicKey(ctx context.Context) ed25519.PublicKey {
	value := ctx.Value(twinPublicKeyID{})
	return value.(ed25519.PublicKey)
}

// TwinID extracts twin id from request
func TwinID(ctx context.Context) uint32 {
	value := ctx.Value(twinKeyID{})
	return value.(uint32)
}

// UserMap implements provision.Twins for the users collections
type UserMap map[uint32]ed25519.PublicKey

// NewUserMap create a new UserMap that uses the users collection
// to find the key
func NewUserMap() UserMap {
	return UserMap{}
}

// AddKeyFromHex adds a user key to map from a hex string
func (u UserMap) AddKeyFromHex(id uint32, key string) error {
	k, err := hex.DecodeString(key)
	if err != nil {
		return err
	}
	u[id] = ed25519.PublicKey(k)
	return nil
}

// GetKey implements interface
func (u UserMap) GetKey(id uint32) ([]byte, error) {
	key, ok := u[id]
	if !ok {
		return nil, fmt.Errorf("unknown user id '%d' in key map", id)
	}
	return key, nil
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)

	object := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(object); err != nil {
		log.Error().Err(err).Msg("failed to encode return object")
	}
}

// NewAuthMiddleware creates a new AuthMiddleware using jwt signed by the caller
func NewAuthMiddleware(twins provision.Twins) mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(twinHeader)
			if len(id) == 0 {
				writeError(w, fmt.Errorf("missing header '%s'", twinHeader))
				return
			}

			twinID, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				writeError(w, errors.Wrapf(err, "wrong twin header '%s'", id))
				return
			}
			pk, err := twins.GetKey(uint32(twinID))
			if err != nil {
				writeError(w, errors.Wrapf(err, "failed to get pk for twin '%d'", twinID))
				return
			}
			request, err := VerifyRequest(pk, r)
			if err != nil {
				writeError(w, err)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, twinKeyID{}, uint32(twinID))
			ctx = context.WithValue(ctx, twinPublicKeyID{}, pk)

			request = request.WithContext(ctx)
			handler.ServeHTTP(w, request)
		})
	}
}
