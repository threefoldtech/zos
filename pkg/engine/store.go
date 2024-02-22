package engine

type Entry struct {
	Payload      []byte
	Dependencies []string
}

type UserID uint32

// abstraction for the internal storage of workloads information
type Store interface {
	SpaceCreate(user UserID, name string) error
	SpaceDelete(user UserID, name string) error
	SpacesList(user UserID) ([]string, error)
	SpaceExists(user UserID, name string) (bool, error)

	ResourceSet(user UserID, space string, entry string, typ string, payload []byte) error
	ResourceGet(user UserID, space string, entry string) (typ string, payload []byte, err error)
	ResourceList(user UserID, space string) ([]string, error)
	ResourceExists(user UserID, space string, name string) (bool, error)

	DependencyAdd(user UserID, space string, entry string, dep string) error
	DependencyRemove(user UserID, space string, entry string, dep string) error

	Scoped(user UserID, space string) ScopedStore
}

type ScopedStore interface {
	ResourceSet(payload []byte) error
	ResourceGet(entry string) (typ string, payload []byte, err error)
	ResourceList() ([]string, error)
	ResourceExists(name string) (exists bool, typ string, err error)
}
