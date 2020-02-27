package phonebook

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/phonebook/types"
)

// UserAPI struct
type UserAPI struct{}

// create user entry point, makes sure name is free for reservation
func (u *UserAPI) create(r *http.Request) (interface{}, mw.Response) {
	var user types.User

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)
	user, err := types.UserCreate(r.Context(), db, user.Name, user.Email, user.Pubkey)
	if err != nil && errors.Is(err, types.ErrUserExists) {
		return nil, mw.Conflict(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return user, nil
}

/*
register
As implemented in threebot. It works as a USER update function. To update
any fields, you need to make sure your payload has an extra "sender_signature_hex"
field that is the signature of the payload using the user private key.

This signature is done on a message that is built as defined by the User.Encode() method
*/
func (u *UserAPI) register(r *http.Request) (interface{}, mw.Response) {
	userID := mux.Vars(r)["user_id"]
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid user id"))
	}

	var payload struct {
		types.User
		Signature string `json:"sender_signature_hex"` // because why not `signature`!
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, mw.BadRequest(err)
	}

	if len(payload.Signature) == 0 {
		return nil, mw.BadRequest(fmt.Errorf("signature is required"))
	}

	signature, err := hex.DecodeString(payload.Signature)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid signature hex"))
	}
	db := mw.Database(r)

	if err := types.UserUpdate(r.Context(), db, schema.ID(id), signature, payload.User); err != nil {
		return nil, mw.Error(err)
	}

	return nil, nil
}

func (u *UserAPI) list(r *http.Request) (interface{}, mw.Response) {
	var (
		name  = r.FormValue("name")
		email = r.FormValue("email")
	)

	var filter types.UserFilter
	if len(name) != 0 {
		filter = filter.WithName(name)
	}
	if len(email) != 0 {
		filter = filter.WithEmail(email)
	}

	db := mw.Database(r)
	cur, err := filter.Find(r.Context(), db, models.PageFromRequest(r))
	if err != nil {
		return nil, mw.Error(err)
	}

	users := []types.User{}
	if err := cur.All(r.Context(), &users); err != nil {
		return nil, mw.Error(err)
	}

	return users, nil
}
