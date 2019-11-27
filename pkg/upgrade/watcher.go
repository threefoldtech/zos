package upgrade

import (
	"context"
	"fmt"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"path"
	"time"
)

/**
Watcher unifies the upgrade pipeline by making sure we can watch
different types of required updates, while always make sure only
one kind of update is applied at a time.

This to prevent updates to step on each other toes.
*/

//EventType of the watcher
type EventType string

const (
	//FList event type
	FList EventType = "flist"
	//Repo event type
	Repo EventType = "repo"
)

//Event interface
type Event interface {
	EventType() EventType
}

//Watcher interface
type Watcher interface {
	Watch(ctx context.Context) (<-chan Event, error)
}

//FListEvent struct
type FListEvent struct {
	flistInfo
}

//EventType of the event
func (f *FListEvent) EventType() EventType {
	return FList
}

//Fqdn return full name of the flist (with repo)
func (f *FListEvent) Fqdn() string {
	return path.Join(f.Repository, f.Name)
}

//TryVersion will try to parse the version from flist
//other wise return empty version 0.0.0
func (f *FListEvent) TryVersion() semver.Version {
	version, _ := f.Version()
	return version
}

//FListSemverWatcher watches a single FList for changes in it's semver
//the semver to change without the flist name itself changes, means
//that this flist is mostly a symlink
type FListSemverWatcher struct {
	FList    string
	Duration time.Duration
	Current  semver.Version

	client hubClient
}

var _ Watcher = &FListSemverWatcher{}

// Watch an flist change in version
// The Event returned by the channel is of concrete type FListEvent
func (w *FListSemverWatcher) Watch(ctx context.Context) (<-chan Event, error) {

	if w.Duration == time.Duration(0) {
		//default delay of 5min
		w.Duration = 600 * time.Second
	}

	info, err := w.client.Info(w.FList)
	if err != nil {
		return nil, err
	}

	version, err := info.Version()
	if err != nil {
		return nil, err
	}

	ch := make(chan Event, 1)

	if version.GT(w.Current) {
		ch <- &FListEvent{
			flistInfo: info,
		}
		w.Current = version
	}

	ticker := time.NewTicker(w.Duration)

	go func() {
		defer ticker.Stop()
		defer close(ch)

		for {
			// wait for next tick
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}

			log.Debug().Str("flist", w.FList).Msg("check updates")

			info, err := w.client.Info(w.FList)
			if err != nil {
				log.Error().Err(err).Str("flist", w.FList).Msg("failed to get flist info")
				continue
			}
			version, err := info.Version()
			if err != nil {
				log.Error().Err(err).Str("flist", w.FList).Msg("failed to get flist version")
				continue
			}

			if version.GT(w.Current) {
				select {
				case ch <- &FListEvent{
					flistInfo: info,
				}:
				case <-ctx.Done():
					return
				}

				w.Current = version
			}
		}
	}()

	return ch, nil
}

type RepoEvent struct {
	Repo  string
	ToAdd []FListEvent
	ToDel []FListEvent
}

func (e *RepoEvent) EventType() EventType {
	return Repo
}

//FListRepoWatcher type
type FListRepoWatcher struct {
	Repo     string
	Current  map[string]FListEvent
	Duration time.Duration

	client hubClient
}

func (w *FListRepoWatcher) list() (map[string]FListEvent, error) {
	packages, err := w.client.List(w.Repo)
	if err != nil {
		return nil, err
	}

	result := make(map[string]FListEvent)
	for _, pkg := range packages {
		flist := FListEvent{pkg}
		result[flist.Fqdn()] = flist
	}

	return result, nil
}

func (w *FListRepoWatcher) diff(packages map[string]FListEvent) (toAdd, toDel []FListEvent) {
	for name, pkg := range packages {
		current, ok := w.Current[name]
		if !ok || pkg.Hash != current.Hash {
			toAdd = append(toAdd, pkg)
		}
	}

	for name := range w.Current {
		current, ok := packages[name]
		if !ok {
			toDel = append(toDel, current)
		}
	}

	return
}

func (w *FListRepoWatcher) Watch(ctx context.Context) (<-chan Event, error) {
	packages, err := w.list()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available packages")
	}

	toAdd, toDel := w.diff(packages)

	ch := make(chan Event, 1)

	if len(toAdd) > 0 || len(toDel) > 0 {
		ch <- &RepoEvent{
			Repo:  w.Repo,
			ToAdd: toAdd,
			ToDel: toDel,
		}

		w.Current = packages
	}

	ticker := time.NewTicker(w.Duration)

	go func() {
		defer close(ch)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}

			packages, err := w.list()
			if err != nil {
				log.Error().Err(err).Str("repo", w.Repo).Msg("failed to list repo flists")
				continue
			}

			toAdd, toDel := w.diff(packages)

			if len(toAdd) > 0 || len(toDel) > 0 {
				select {
				case ch <- &RepoEvent{
					Repo:  w.Repo,
					ToAdd: toAdd,
					ToDel: toDel,
				}:
				case <-ctx.Done():
					return
				}

				w.Current = packages
			}
		}

	}()

	return ch, fmt.Errorf("not implemented")
}
