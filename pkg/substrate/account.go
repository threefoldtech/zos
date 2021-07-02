package substrate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/vedhavyas/go-subkey"
	subkeyEd25519 "github.com/vedhavyas/go-subkey/ed25519"
)

// AccountID type
type AccountID types.AccountID

//PublicKey gets public key from account id
func (a AccountID) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(a[:])
}

// keyringPairFromSecret creates KeyPair based on seed/phrase and network
// Leave network empty for default behavior
func keyringPairFromSecret(seedOrPhrase string, network uint8) (signature.KeyringPair, error) {
	scheme := subkeyEd25519.Scheme{}
	kyr, err := subkey.DeriveKeyPair(scheme, seedOrPhrase)

	if err != nil {
		return signature.KeyringPair{}, err
	}

	ss58Address, err := kyr.SS58Address(network)
	if err != nil {
		return signature.KeyringPair{}, err
	}

	var pk = kyr.Public()

	return signature.KeyringPair{
		URI:       seedOrPhrase,
		Address:   ss58Address,
		PublicKey: pk,
	}, nil
}

var (
	errAccountNotFound = fmt.Errorf("account not found")
)

/*
curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"kycSignature": "", "data": {"name": "", "email": ""}, "substrateAccountID": "5DAprR72N6s7AWGwN7TzV9MyuyGk9ifrq8kVxoXG9EYWpic4"}' \
  https://api.substrate01.threefold.io/activate
*/

func (s *Substrate) activateAccount(identity signature.KeyringPair) error {
	const url = "https://api.substrate01.threefold.io/activate"

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(map[string]string{
		"substrateAccountID": identity.Address,
	})

	response, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return errors.Wrap(err, "failed to call activation service")
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusConflict {
		// it went fine.
		return nil
	}

	return fmt.Errorf("failed to activate account: %s", response.Status)
}

// EnsureAccount makes sure account is available on blockchain
// if not, it uses activation service to create one
func (s *Substrate) EnsureAccount(sk ed25519.PrivateKey) (info types.AccountInfo, err error) {
	identity, err := Identity(sk)
	if err != nil {
		return
	}

	info, err = s.getAccount(identity, s.meta)
	if errors.Is(err, errAccountNotFound) {
		// account activation
		if err = s.activateAccount(identity); err != nil {
			return
		}

		// after activation this can take up to 10 seconds
		// before the account is actually there !

		exp := backoff.NewExponentialBackOff()
		exp.MaxElapsedTime = 10 * time.Second
		exp.MaxInterval = 3 * time.Second

		err = backoff.Retry(func() error {
			info, err = s.getAccount(identity, s.meta)
			return err
		}, exp)

		return
	}

	return

}

// Identity derive the correct substrate identity from ed25519 key
func Identity(sk ed25519.PrivateKey) (signature.KeyringPair, error) {
	str := types.HexEncodeToString(sk.Seed())
	return keyringPairFromSecret(str, 42)
	// because 42 is the answer to life the universe and everything
	// no, seriously, don't change it, it has to be 42.
}

func (s *Substrate) getAccount(identity signature.KeyringPair, meta *types.Metadata) (info types.AccountInfo, err error) {
	key, err := types.CreateStorageKey(meta, "System", "Account", identity.PublicKey, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create storage key")
		return
	}

	ok, err := s.cl.RPC.State.GetStorageLatest(key, &info)
	if err != nil || !ok {
		if !ok {
			return info, errAccountNotFound
		}

		return
	}

	return
}

// GetAccount gets account info with secure key
func (s *Substrate) GetAccount(sk ed25519.PrivateKey) (info types.AccountInfo, err error) {
	identity, err := Identity(sk)
	if err != nil {
		return
	}
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return info, err
	}

	return s.getAccount(identity, meta)
}