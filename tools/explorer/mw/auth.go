package mw

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/phonebook/types"
	"github.com/zaibon/httpsig"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoKeyGetter struct {
	db *mongo.Database
}

func (m mongoKeyGetter) GetKey(id string) interface{} {
	ctx := context.TODO()

	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil
	}

	f := types.UserFilter{}

	f = f.WithID(schema.ID(uid))

	col := m.db.Collection(types.UserCollection)
	result := col.FindOne(ctx, f, options.FindOne())

	err = result.Err()
	if err != nil {
		return nil
	}

	user := types.User{}
	if err = result.Decode(&user); err != nil {
		return nil
	}

	pk, err := hex.DecodeString(user.Pubkey)
	if err != nil {
		return nil
	}
	return pk
}

// requiredHeaders are the parameters to be used to generated the http signature
var requiredHeaders = []string{"(created)", "date", "threebot-id"}

func AuthMiddleware(db *mongo.Database, h http.Handler) http.Handler {
	kg := mongoKeyGetter{db}
	verifier := httpsig.NewVerifier(kg)
	verifier.SetRequiredHeaders(requiredHeaders)

	return requireSignature(h, verifier)
}

func requireSignature(h http.Handler, v *httpsig.Verifier) (
	out http.Handler) {

	var challengeParams []string
	if headers := v.RequiredHeaders(); len(headers) > 0 {
		challengeParams = append(challengeParams,
			fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challengeParams) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challengeParams, ", "))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := v.Verify(req)
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
		h.ServeHTTP(w, req)
	})
}
