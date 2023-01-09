package direct

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/threefoldtech/substrate-client"
)

const CustomSigning = "RMB"

var (
	_ jwt.SigningMethod = (*RmbSigner)(nil)
)

type RmbSigner struct{}

func (s *RmbSigner) Verify(signingString, signature string, key interface{}) error {
	panic("unimplemented")
}

func (s *RmbSigner) Sign(signingString string, key interface{}) (string, error) {
	fmt.Printf("Sig string: %s\n", signingString)
	identity, ok := key.(substrate.Identity)
	if !ok {
		return "", fmt.Errorf("invalid key expecting substrate identity")
	}

	signature, err := Sign(identity, []byte(signingString))
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *RmbSigner) Alg() string {
	return "RS512"
}

func NewJWT(id uint32, identity substrate.Identity) (string, error) {
	token := jwt.NewWithClaims(&RmbSigner{}, jwt.MapClaims{
		"sub": id,
		"iat": time.Now().Unix(),
		"exp": 600,
	})

	return token.SignedString(identity)
}

func Sign(signer substrate.Identity, input []byte) ([]byte, error) {
	signature, err := signer.Sign(input)
	if err != nil {
		return nil, err
	}
	withType := make([]byte, len(signature)+1)

	withType[0] = signer.Type()[0] // edIdentity will return e, while sr will be s
	copy(withType[1:], signature)
	return withType, nil
}
