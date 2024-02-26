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
	ResourceExists(user UserID, space string, name string) (exists bool, typ string, err error)
	ResourceDelete(user UserID, space string, name string) error

	DependencyAdd(user UserID, space string, entry string, dep string) error
	DependencyRemove(user UserID, space string, entry string, dep string) error

	Scoped(user UserID, space string, entry, typ string) ScopedStore
}

type ScopedStore interface {
	ResourceSet(payload []byte) error
	ResourceGet(entry string) (typ string, payload []byte, err error)
	ResourceList() ([]string, error)
	ResourceExists(name string) (exists bool, typ string, err error)
}

type objectBucket struct {
	id   string
	typ  string
	data []byte
}

type spaceBucket struct {
	objects map[string]objectBucket
}
type userBucket struct {
	spaces map[string]*spaceBucket
}

// MemStore usually used for testing
type MemStore struct {
	users map[UserID]*userBucket
}

func (s *MemStore) SpaceCreate(user UserID, name string) error {
	bucket := s.users[user]

	if bucket == nil {
		bucket = &userBucket{
			spaces: make(map[string]*spaceBucket),
		}
		s.users[user] = bucket
	}

	bucket.spaces[name] = &spaceBucket{
		objects: make(map[string]objectBucket),
	}

	s.users[user] = bucket
	return nil
}

func (s *MemStore) SpaceDelete(user UserID, name string) error {
	bucket := s.users[user]

	if bucket == nil {
		return nil
	}

	delete(bucket.spaces, name)

	return nil
}

func (s *MemStore) SpacesList(user UserID) ([]string, error) {
	bucket := s.users[user]

	if bucket == nil {
		return nil, nil
	}
	var spaces []string
	for space := range bucket.spaces {
		spaces = append(spaces, space)
	}

	return spaces, nil
}

func (s *MemStore) SpaceExists(user UserID, name string) (bool, error) {
	_, ok := s.getSpace(user, name)
	return ok, nil
}

func (s *MemStore) getSpace(user UserID, name string) (*spaceBucket, bool) {
	bucket := s.users[user]

	if bucket == nil {
		return nil, false
	}

	space, ok := bucket.spaces[name]
	return space, ok
}

func (s *MemStore) ResourceSet(user UserID, space string, entry string, typ string, payload []byte) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return ErrSpaceNotFound
	}

	bkt.objects[entry] = objectBucket{
		id:   entry,
		typ:  typ,
		data: payload,
	}

	return nil
}
func (s *MemStore) ResourceGet(user UserID, space string, entry string) (typ string, payload []byte, err error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return typ, nil, ErrSpaceNotFound
	}

	obj, ok := bkt.objects[entry]
	if !ok {
		return typ, nil, ErrObjectNotFound
	}

	return obj.typ, obj.data, nil
}

func (s *MemStore) ResourceList(user UserID, space string) ([]string, error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return nil, ErrSpaceNotFound
	}

	var objs []string
	for obj := range bkt.objects {
		objs = append(objs, obj)
	}

	return objs, nil

}
func (s *MemStore) ResourceExists(user UserID, space string, name string) (exists bool, typ string, err error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return false, typ, ErrSpaceNotFound
	}

	obj, ok := bkt.objects[name]
	if !ok {
		return false, typ, nil
	}

	return true, obj.typ, nil
}

func (s *MemStore) ResourceDelete(user UserID, space string, name string) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return nil
	}

	delete(bkt.objects, name)
	return nil
}

func (s *MemStore) DependencyAdd(user UserID, space string, entry string, dep string) error {
	panic("not implemented")
}
func (s *MemStore) DependencyRemove(user UserID, space string, entry string, dep string) error {
	panic("not implemented")
}

func (s *MemStore) Scoped(user UserID, space string, entry, typ string) ScopedStore {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		//TODO should not happen
		panic("space does not exist")
	}

	return &scopedMemStore{
		space: bkt,
		id:    entry,
		typ:   typ,
	}
}

type scopedMemStore struct {
	space   *spaceBucket
	id, typ string
}

func (s *scopedMemStore) ResourceSet(payload []byte) error {
	s.space.objects[s.id] = objectBucket{
		id:   s.id,
		typ:  s.typ,
		data: payload,
	}

	return nil
}

func (s *scopedMemStore) ResourceGet(entry string) (typ string, payload []byte, err error) {
	obj, ok := s.space.objects[entry]
	if !ok {
		return typ, payload, ErrObjectNotFound
	}
	return obj.typ, obj.data, nil
}

func (s *scopedMemStore) ResourceList() ([]string, error) {
	var names []string
	for n := range s.space.objects {
		names = append(names, n)
	}

	return names, nil
}
func (s *scopedMemStore) ResourceExists(name string) (exists bool, typ string, err error) {
	obj, ok := s.space.objects[name]
	if ok {
		return true, obj.typ, nil
	}

	return false, typ, nil
}
