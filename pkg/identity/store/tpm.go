package store

import "crypto/ed25519"

type TPMStore struct{}

var _ Store = (*TPMStore)(nil)

// Get returns the key from the store
func (t *TPMStore) Get() (ed25519.PrivateKey, error) {
	panic("unimplemented")
}

// Updates, or overrides the current key
func (t *TPMStore) Set(key ed25519.PrivateKey) error {
	panic("unimplemented")
}

// Check if key there is a key stored in the
// store
func (t *TPMStore) Exists() (bool, error) {
	panic("unimplemented")
}

// Destroys the key
func (t *TPMStore) Annihilate() error {
	panic("unimplemented")
}
