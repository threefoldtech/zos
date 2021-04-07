package gridtypes

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrWorkloadNotFound error
	ErrWorkloadNotFound = fmt.Errorf("workload not found")
)

// https://github.com/threefoldtech/vgrid/blob/main/zosreq/req_deployment_root.v

// WorkloadGetter is used to get a workload by name inside
// the deployment context. Mainly used to validate dependency
type WorkloadGetter interface {
	Get(name string) (*WorkloadWithID, error)
}

// WorkloadByTypeGetter is used to get a list of workloads
// of specific type from a workload container like a deployment
type WorkloadByTypeGetter interface {
	ByType(typ WorkloadType) []*WorkloadWithID
}

// Deployment structure
type Deployment struct {
	Version              int                  `json:"version"`
	TwinID               uint32               `json:"twin_id"`
	DeploymentID         uint32               `json:"deployment_id"`
	Metadata             string               `json:"metadata"`
	Description          string               `json:"description"`
	Expiration           Timestamp            `json:"expiration"`
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	Workloads            []Workload           `json:"workloads"`
}

// WorkloadWithID wrapper around workload type
// that holds the global workload ID
type WorkloadWithID struct {
	*Workload
	ID WorkloadID
}

// SignatureRequest struct
type SignatureRequest struct {
	TwinID   uint32 `json:"twin_id"`
	Required bool   `json:"required"`
	Weight   uint   `json:"weight"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequest) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", r.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", r.Required); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", r.Weight); err != nil {
		return err
	}

	return nil
}

// Signature struct
type Signature struct {
	TwinID    uint32 `json:"twin_id"`
	Signature string `json:"signature"`
}

// SignatureRequirement struct
type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired int                `json:"weight_required"`
	Signatures     []Signature        `json:"signatures"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequirement) Challenge(w io.Writer) error {
	for _, request := range r.Requests {
		if err := request.Challenge(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d", r.WeightRequired); err != nil {
		return err
	}

	return nil
}

// Challenge computes challenge for SignatureRequest
func (d *Deployment) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", d.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.DeploymentID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Metadata); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Description); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.Expiration); err != nil {
		return err
	}

	for _, wl := range d.Workloads {
		if err := wl.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}

// Valid validates deployment structure
func (d *Deployment) Valid() error {
	names := make(map[string]struct{})
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		name := wl.Name
		if err := IsValidName(name); err != nil {
			return errors.Wrapf(err, "name '%s' is invalid", name)
		}

		if _, ok := names[name]; ok {
			return fmt.Errorf("multiple workloads with the same name '%s'", name)
		}
		names[name] = struct{}{}

		if err := wl.Valid(d); err != nil {
			return err
		}
	}

	return nil
}

// Get a workload by name
func (d *Deployment) Get(name string) (*WorkloadWithID, error) {
	if err := IsValidName(name); err != nil {
		return nil, err
	}

	for i := range d.Workloads {
		wl := &d.Workloads[i]
		id, _ := NewWorkloadID(d.TwinID, d.DeploymentID, name)
		if wl.Name == name {
			return &WorkloadWithID{
				Workload: wl,
				ID:       id,
			}, nil
		}
	}

	return nil, errors.Wrapf(ErrWorkloadNotFound, "no workload with name '%s'", name)
}

// GetType gets a reservation by name if only of the correct type.
func (d *Deployment) GetType(name string, typ WorkloadType) (*WorkloadWithID, error) {
	wl, err := d.Get(name)
	if err != nil {
		return nil, err
	}
	if wl.Type != typ {
		return nil, fmt.Errorf("workload '%s' of wrong type", name)
	}

	return wl, nil
}

// ByType gets all workloads from this reservation by type.
func (d *Deployment) ByType(typ WorkloadType) []*WorkloadWithID {
	var results []*WorkloadWithID
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Type == typ {
			id, err := NewWorkloadID(d.TwinID, d.DeploymentID, wl.Name)
			if err != nil {
				log.Warn().Err(err).Msg("deployment has invalid name. please run validation")
				continue
			}

			results = append(
				results,
				&WorkloadWithID{
					Workload: wl,
					ID:       id,
				},
			)
		}
	}

	return results
}

// Upgrade validates n as an updated version of d, and return an Upgrade description
// for the steps that the node needs to take.
func (d *Deployment) Upgrade(n *Deployment) (*Upgrade, error) {
	if err := n.Valid(); err != nil {
		return nil, errors.Wrap(err, "new deployment is invalid")
	}

	if d.TwinID != n.TwinID || d.DeploymentID != n.DeploymentID {
		return nil, fmt.Errorf("cannot change deployment or twin id")
	}

	expected := d.Version + 1
	if expected != n.Version {
		return nil, fmt.Errorf("expecting deployment version %d, got %d", expected, n.Version)
	}

	current := make(map[string]*Workload)
	for i := range d.Workloads {
		wl := &d.Workloads[i]

		current[wl.Name] = wl
	}

	update := make([]*WorkloadWithID, 0)
	add := make([]*WorkloadWithID, 0)
	remove := make([]*WorkloadWithID, 0)

	for i := range n.Workloads {
		l := &n.Workloads[i]
		id, err := NewWorkloadID(n.TwinID, n.DeploymentID, l.Name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get build workload ID")
		}
		wl := &WorkloadWithID{
			Workload: l,
			ID:       id,
		}
		older, ok := current[wl.Name]

		if !ok {
			if wl.Version == expected {
				// newly added workload
				add = append(add, wl)
			} else {
				return nil, fmt.Errorf("invalid version number for workload '%s' expected '%d'", wl.Name, expected)
			}
		} else {
			// modifying a type.
			if older.Type != wl.Type {
				return nil, fmt.Errorf("cannot change workload type '%s'", wl.Name)
			}

			// older version that exists.
			// so
			if wl.Version == expected {
				// should added to 'update' pile
				update = append(update, wl)
			}
			// other wise. we leave it untouched
		}

		// in both cases, we remove this from the current list
		delete(current, wl.Name)
	}

	for _, wl := range current {
		id, _ := NewWorkloadID(d.TwinID, d.DeploymentID, wl.Name)

		remove = append(remove, &WorkloadWithID{
			Workload: wl,
			ID:       id,
		})
	}

	return &Upgrade{ToAdd: add, ToUpdate: update, ToRemove: remove}, nil
}

// Upgrade procedure structure
type Upgrade struct {
	ToAdd    []*WorkloadWithID
	ToUpdate []*WorkloadWithID
	ToRemove []*WorkloadWithID
}
