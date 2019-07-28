package modules

// Identity defines zbus interface for identity daemon.
type Identity interface {
	// NodeID returns the node id (public key)
	NodeID() string

	// Encrypt data with node private key. Data must be decrypted with node public key (node id)
	// at the other end.
	Encrypt(data []byte) ([]byte, error)

	// Decrypt data with node private key. Data must have been decrypted with node public key (node id)
	Decrypt(data []byte) ([]byte, error)

	// Sign
	Sign(data []byte) ([]byte, error)

	// Verify
	Verify(data []byte) error
}
