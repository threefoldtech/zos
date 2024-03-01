package engine

import "slices"

type Record struct {
	ID      string
	Version uint
	Type    string
	Data    []byte
	Masters []string
}

type UserID uint32

// abstraction for the internal storage of workloads information
type Store interface {
	SpaceCreate(user UserID, name string) error
	SpaceDelete(user UserID, name string) error
	SpacesList(user UserID) ([]string, error)
	SpaceExists(user UserID, name string) (bool, error)

	RecordSet(user UserID, space string, record Record) error
	RecordGet(user UserID, space string, id string) (record Record, err error)
	RecordList(user UserID, space string) ([]string, error)
	RecordExists(user UserID, space string, id string) (exists bool, typ string, err error)
	RecordDelete(user UserID, space string, id string) error

	MasterAdd(user UserID, space string, id string, master string) error
	MasterRemove(user UserID, space string, id string, master string) error
	IsSlave(user UserID, space, id string) (bool, error)

	Scoped(user UserID, space string, entry, typ string) ScopedStore
}

type ScopedStore interface {
	RecordSet(payload []byte) error
	RecordGet(entry string) (record Record, err error)
	RecordList() ([]string, error)
	RecordExists(id string) (exists bool, typ string, err error)
}

type spaceBucket struct {
	objects map[string]Record
}

type userBucket struct {
	spaces map[string]*spaceBucket
}

// MemStore usually used for testing
type MemStore struct {
	users map[UserID]*userBucket
}

var _ Store = (*MemStore)(nil)

func NewMemStore() *MemStore {
	return &MemStore{
		users: make(map[UserID]*userBucket),
	}
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
		objects: make(map[string]Record),
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

func (s *MemStore) RecordSet(user UserID, space string, record Record) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return ErrSpaceNotFound
	}

	bkt.objects[record.ID] = record

	return nil
}
func (s *MemStore) RecordGet(user UserID, space string, entry string) (record Record, err error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return record, ErrSpaceNotFound
	}

	record, ok = bkt.objects[entry]
	if !ok {
		return record, ErrObjectDoesNotExist
	}

	return record, nil
}

func (s *MemStore) RecordList(user UserID, space string) ([]string, error) {
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
func (s *MemStore) RecordExists(user UserID, space string, name string) (exists bool, typ string, err error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return false, typ, ErrSpaceNotFound
	}

	obj, ok := bkt.objects[name]
	if !ok {
		return false, typ, nil
	}

	return true, obj.Type, nil
}

func (s *MemStore) RecordDelete(user UserID, space string, name string) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return nil
	}

	delete(bkt.objects, name)
	return nil
}

func (s *MemStore) MasterAdd(user UserID, space string, entry string, dep string) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return ErrSpaceNotFound
	}

	record, ok := bkt.objects[entry]
	if !ok {
		return ErrObjectDoesNotExist
	}

	record.Masters = append(record.Masters, dep)
	slices.Sort(record.Masters)
	record.Masters = slices.Compact(record.Masters)
	bkt.objects[entry] = record

	return nil
}

func (s *MemStore) IsSlave(user UserID, space string, entry string) (bool, error) {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return false, ErrSpaceNotFound
	}

	record, ok := bkt.objects[entry]
	if !ok {
		return false, ErrObjectDoesNotExist
	}

	return len(record.Masters) > 0, nil
}

func (s *MemStore) MasterRemove(user UserID, space string, entry string, dep string) error {
	bkt, ok := s.getSpace(user, space)
	if !ok {
		return ErrSpaceNotFound
	}

	record, ok := bkt.objects[entry]
	if !ok {
		return ErrObjectDoesNotExist
	}

	masters := record.Masters[:0]
	for _, m := range record.Masters {
		if m == dep {
			continue
		}
		masters = append(masters, m)
	}

	record.Masters = masters
	bkt.objects[entry] = record

	return nil
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

func (s *scopedMemStore) RecordSet(payload []byte) error {
	s.space.objects[s.id] = Record{
		ID:   s.id,
		Type: s.typ,
		Data: payload,
	}

	return nil
}

func (s *scopedMemStore) RecordGet(entry string) (record Record, err error) {
	record, ok := s.space.objects[entry]
	if !ok {
		return record, ErrObjectDoesNotExist
	}
	return record, nil
}

func (s *scopedMemStore) RecordList() ([]string, error) {
	var names []string
	for n := range s.space.objects {
		names = append(names, n)
	}

	return names, nil
}
func (s *scopedMemStore) RecordExists(name string) (exists bool, typ string, err error) {
	obj, ok := s.space.objects[name]
	if ok {
		return true, obj.Type, nil
	}

	return false, obj.Type, nil
}
