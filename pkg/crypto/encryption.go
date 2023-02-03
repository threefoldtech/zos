package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/agl/ed25519/extra25519"
	box "github.com/whs/nacl-sealed-box"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/secretbox"
)

// Encrypt encrypts msg with a cure25519 public key derived from an ed25519 public key
func Encrypt(msg []byte, pk ed25519.PublicKey) ([]byte, error) {
	curvePub := PublicKeyToCurve25519(pk)
	return box.Seal(msg, &curvePub)
}

// Decrypt decrypts msg with a cure25519 private key derived from an ed25519 private key
func Decrypt(msg []byte, sk ed25519.PrivateKey) ([]byte, error) {
	curvePriv := PrivateKeyToCurve25519(sk)
	curvePub := PublicKeyToCurve25519(sk.Public().(ed25519.PublicKey))

	if len(msg) < 48 {
		return nil, fmt.Errorf("invalid cipher text too short")
	}

	return box.Open(msg, &curvePub, &curvePriv)
}

// PrivateKeyToCurve25519 converts an ed25519 private key into a corresponding curve25519 private key
// such that the resulting curve25519 public key will equal the result from PublicKeyToCurve25519.
func PrivateKeyToCurve25519(sk ed25519.PrivateKey) [32]byte {
	curvePriv := [32]byte{}
	edPriv := [ed25519.PrivateKeySize]byte{}
	copy(edPriv[:], sk)
	extra25519.PrivateKeyToCurve25519(&curvePriv, &edPriv)
	return curvePriv
}

// PublicKeyToCurve25519 converts an Ed25519 public key into the curve25519 public
//
//	key that would be generated from the same private key.
func PublicKeyToCurve25519(pk ed25519.PublicKey) [32]byte {
	curvePub := [32]byte{}
	edPriv := [ed25519.PublicKeySize]byte{}
	copy(edPriv[:], pk)
	extra25519.PublicKeyToCurve25519(&curvePub, &edPriv)
	return curvePub
}

// DecryptECDH decrypt aes encrypted msg using a shared key derived from sk and pk using Elliptic curve Diffie Helman algorithm
func DecryptECDH(msg []byte, sk ed25519.PrivateKey, pk ed25519.PublicKey) ([]byte, error) {

	key, err := sharedSecret(sk, pk)
	if err != nil {
		return nil, err
	}

	var nonce [24]byte
	copy(nonce[:], msg[:24])

	decrypted, ok := secretbox.Open(nil, msg[24:], &nonce, &key)
	if !ok {
		return nil, fmt.Errorf("decryption error")
	}

	return decrypted, nil
}

// EncryptECDH aes encrypt msg using a shared key derived from sk and pk using Elliptic curve Diffie Helman algorithm
// the nonce if prepended to the encrypted message
func EncryptECDH(msg []byte, sk ed25519.PrivateKey, pk ed25519.PublicKey) ([]byte, error) {

	key, err := sharedSecret(sk, pk)
	if err != nil {
		return nil, err
	}

	var nonce [24]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	encrypted := secretbox.Seal(nonce[:], msg, &nonce, &key)
	return encrypted, nil

}

func sharedSecret(sk ed25519.PrivateKey, pk ed25519.PublicKey) ([32]byte, error) {
	private := PrivateKeyToCurve25519(sk)
	public := PublicKeyToCurve25519(pk)

	shareSecret, err := curve25519.X25519(private[:], public[:])
	if err != nil {
		return [32]byte{}, err
	}

	return blake2b.Sum256(shareSecret), nil
}
