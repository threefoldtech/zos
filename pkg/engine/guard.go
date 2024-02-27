package engine

import "sync"

// Guard is a kind of shared lock on a single resource.
// Once a guard is retrieved by a call to AccessGuard.Enter()
// the guard must be "Exited" (defer guard.Exit()) during the time
// the guard is acquired, a guard can be locked and unlocked as many
// time as needed.
type Guard interface {
	sync.Locker
	RLock()
	RUnlock()
	// drop the guard it's now illegal to
	// try to acquire the lock again
	Exit()
}

// accessGuard is used internally to
// manage exclusive access to objects
type accessGuard struct {
	id     string
	parent *AccessGuard
	sync.RWMutex
	count uint32
}

func (g *accessGuard) Exit() {
	g.parent.exit(g.id)
}

type AccessGuard struct {
	guards map[string]*accessGuard
	m      sync.Mutex
}

func NewAccessGuard() *AccessGuard {
	return &AccessGuard{
		guards: map[string]*accessGuard{},
	}
}

func (g *AccessGuard) Enter(id string) Guard {
	g.m.Lock()
	defer g.m.Unlock()

	guard, ok := g.guards[id]
	if !ok {
		guard = &accessGuard{
			id:     id,
			parent: g,
		}
	}
	guard.count += 1
	return guard
}

func (g *AccessGuard) exit(id string) {
	g.m.Lock()
	defer g.m.Unlock()

	guard := g.guards[id]
	guard.count -= 1
	if guard.count == 0 {
		delete(g.guards, id)
	}
}
