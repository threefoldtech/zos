package mw

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/zaibon/httpsig"
)

type usePublicKeyID struct{}
type userKeyID struct{}

// UserPublicKey extracts user public key from request
func UserPublicKey(ctx context.Context) ed25519.PublicKey {
	value := ctx.Value(usePublicKeyID{})
	return value.(ed25519.PublicKey)
}

// UserID extracts user id from request
func UserID(ctx context.Context) gridtypes.ID {
	value := ctx.Value(userKeyID{})
	return value.(gridtypes.ID)
}

// UserMap implements httpsig.KeyGetter for the users collections
type UserMap map[gridtypes.ID]ed25519.PublicKey

// NewUserMap create a httpsig.KeyGetter that uses the users collection
// to find the key
func NewUserMap() provision.Users {
	return UserMap{}
}

// GetKey implements httpsig.KeyGetter
func (u UserMap) GetKey(id gridtypes.ID) ed25519.PublicKey {
	key, _ := u[id]
	return key
}

// requiredHeaders are the parameters to be used to generated the http signature
var requiredHeaders = []string{"(created)", "date"}

// AuthMiddleware implements https://tools.ietf.org/html/draft-cavage-http-signatures-12
// authentication scheme as an HTTP middleware
type AuthMiddleware struct {
	verifier  *httpsig.Verifier
	challenge string
}

type keyGetter struct {
	users provision.Users
}

func (k *keyGetter) GetKey(id string) interface{} {
	return k.users.GetKey(gridtypes.ID(id))
}

// NewAuthMiddleware creates a new AuthMiddleware using the v httpsig.Verifier
func NewAuthMiddleware(users provision.Users) mux.MiddlewareFunc {
	verifier := httpsig.NewVerifier(&keyGetter{users})
	verifier.SetRequiredHeaders(requiredHeaders)
	var challengeParams []string
	if headers := verifier.RequiredHeaders(); len(headers) > 0 {
		challengeParams = append(challengeParams,
			fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challengeParams) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challengeParams, ", "))
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userID, err := verifier.Verify(req)
			if err != nil {
				w.Header()["WWW-Authenticate"] = []string{challenge}
				w.WriteHeader(http.StatusUnauthorized)

				log.Error().Err(err).Msgf("unauthorized access to %s", req.URL.Path)

				object := struct {
					Error string `json:"error"`
				}{
					Error: errors.Wrap(err, "unauthorized access").Error(),
				}
				if err := json.NewEncoder(w).Encode(object); err != nil {
					log.Error().Err(err).Msg("failed to encode return object")
				}
				return
			}

			pk := users.GetKey(gridtypes.ID(userID))
			ctx := req.Context()
			ctx = context.WithValue(ctx, userKeyID{}, gridtypes.ID(userID))
			ctx = context.WithValue(ctx, usePublicKeyID{}, pk)

			handler.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
