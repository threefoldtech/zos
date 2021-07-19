package gridtypes

import (
	"crypto/ed25519"
	"crypto/md5"
	"encoding/hex"
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
	Get(name Name) (*WorkloadWithID, error)
}

// WorkloadByTypeGetter is used to get a list of workloads
// of specific type from a workload container like a deployment
type WorkloadByTypeGetter interface {
	ByType(typ WorkloadType) []*WorkloadWithID
}

// KeyGetter interface to get key by twin ids
type KeyGetter interface {
	GetKey(twin uint32) (ed25519.PublicKey, error)
}

// Deployment structure
type Deployment struct {
	// Version must be set to 0 on deployment creation. And then it has to
	// be incremented with each call to update.
	Version int `json:"version"`
	// TwinID is the id of the twin sendign the deployment. A twin then can only
	// `get` status about deployments he owns.
	TwinID uint32 `json:"twin_id"`
	// ContractID the contract must be "pre created" on substrate before the deployment is
	// sent to the node. The node will then validate that this deployment hash, will match the
	// hash attached to this contract.
	// the flow should go as follows:
	// - fill in ALL deployment details (metadata, and workloads)
	// - calculate the deployment hash (by calling ChallengeHash method)
	// - create the contract with the right hash
	// - set the contract id on the deployment object
	// - send deployment to node.
	ContractID uint64 `json:"contract_id"`
	// Metadata is user specific meta attached to deployment, can be used to link this
	// deployment to other external systems for automation
	Metadata string `json:"metadata"`
	// Description is human readable description of the deployment
	Description string `json:"description"`
	// Expiration [deprecated] is not used
	Expiration Timestamp `json:"expiration"`
	// SignatureRequirement specifications
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	// Workloads is a list of workloads associated with this deployment
	Workloads []Workload `json:"workloads"`
}

// SetError sets an error on ALL workloads. this is mostly
// an error caused by validation AFTTER the deployment was initially accepted
func (d *Deployment) SetError(err error) {
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		wl.Result.State = StateError
		wl.Result.Error = err.Error()
	}
}

// WorkloadWithID wrapper around workload type
// that holds the global workload ID
// Note: you never need to construct this manually
type WorkloadWithID struct {
	*Workload
	ID WorkloadID
}

// SignatureRequest struct a signature request of a twin
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

// SignatureRequirement struct describes the signatures that are needed to be valid
// for the node to accept the deployment
// for example
// SignatureRequirement{
// 	WeightRequired: 1,
// 	Requests: []gridtypes.SignatureRequest{
// 		{
// 			TwinID: twinID,
// 			Weight: 1,
// 		},
// 	},
// }
// basically states that a total signature weight of 1 is required for the node to accept
// the deployment.
// the list of acceptable signatures is one from twin with `twinID` and his signature weight is 1
// So, in this example this twin signature is enough.
// You can build a more sophisticated signature request to allow multiple twins to sign for example
// SignatureRequirement{
// 	WeightRequired: 2,
// 	Requests: []gridtypes.SignatureRequest{
// 		{
// 			TwinID: Twin1,
// 			Weight: 1,
// 		},
// 		{
// 			TwinID: Twin2,
// 			Weight: 1,
// 		},
// 		{
// 			TwinID: Twin3,
// 			Required: true,
// 			Weight: 1,
// 		},
// 	},
// },
// this means that twin3 must sign + one of either (twin1 or twin2) to have the right signature weight
type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired uint               `json:"weight_required"`
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

// ChallengeHash computes the hash of the deployment. The hash is needed for the following
// - signing the deployment (done automatically by call to "Sign")
// - contract creation, the contract need to be created by this hash exactly BEFORE sending the
//   deployment to the node
// - node verifies the hash to make sure it matches hash of the contract
func (d *Deployment) ChallengeHash() ([]byte, error) {
	hash := md5.New()
	if err := d.Challenge(hash); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// Challenge computes challenge for SignatureRequest
func (d *Deployment) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", d.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.TwinID); err != nil {
		return err
	}

	// ContractID is intentionally removed from the hash calculations
	// because the contract is created on the blockchain first (requires this hash)
	// then set on the deployment before send to the node
	//
	// if _, err := fmt.Fprintf(w, "%d", d.ContractID); err != nil {
	// 	return err
	// }

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

	if err := d.SignatureRequirement.Challenge(w); err != nil {
		return err
	}

	return nil
}

// Valid validates deployment structure
func (d *Deployment) Valid() error {
	names := make(map[Name]struct{})
	current := d.Version
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Version > current {
			return fmt.Errorf("workload '%s' version '%d' cannot be higher than deployment version '%d'", wl.Name, wl.Version, current)
		}

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

// Sign adds a signature to deployment given twin id
func (d *Deployment) Sign(twin uint32, sk ed25519.PrivateKey) error {
	message, err := d.ChallengeHash()
	if err != nil {
		return err
	}
	signatureBytes := ed25519.Sign(sk, message)
	signature := hex.EncodeToString(signatureBytes)
	for i := range d.SignatureRequirement.Signatures {
		sig := &d.SignatureRequirement.Signatures[i]
		// update
		if sig.TwinID == twin {
			sig.Signature = signature
			return nil
		}
	}

	d.SignatureRequirement.Signatures = append(
		d.SignatureRequirement.Signatures, Signature{
			TwinID:    twin,
			Signature: signature,
		})
	return nil
}

// Verify verifies user signatures is mainly used by the node
// to verify that all attached signatures are valid.
func (d *Deployment) Verify(getter KeyGetter) error {
	message, err := d.ChallengeHash()
	if err != nil {
		return err
	}

	requirements := &d.SignatureRequirement
	get := func(twin uint32) (Signature, bool) {
		for _, sig := range requirements.Signatures {
			if sig.TwinID == twin {
				return sig, true
			}
		}
		return Signature{}, false
	}

	originatorFound := false
	var weight uint
	for _, request := range requirements.Requests {
		if request.TwinID == d.TwinID {
			originatorFound = true
			request.Required = true // we force that this is a required signature
		}

		signature, ok := get(request.TwinID)
		if !ok && request.Required {
			return fmt.Errorf("missing required signature for twin '%d'", request.TwinID)
		}

		pk, err := getter.GetKey(request.TwinID)
		if err != nil {
			return errors.Wrapf(err, "failed to get public key for twin '%d'", request.TwinID)
		}

		bytes, err := hex.DecodeString(signature.Signature)
		if err != nil {
			return errors.Wrap(err, "invalid signature")
		}

		if !ed25519.Verify(pk, message, bytes) {
			return fmt.Errorf("failed to verify signature")
		}
		weight += request.Weight
	}

	if !originatorFound {
		return fmt.Errorf("originator twin id must be in the signature requests")
	}

	if weight < requirements.WeightRequired {
		return fmt.Errorf("required signature weight is not reached")
	}

	return nil
}

// Get a workload by name
func (d *Deployment) Get(name Name) (*WorkloadWithID, error) {
	if err := IsValidName(name); err != nil {
		return nil, err
	}

	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Name == name {
			id, _ := NewWorkloadID(d.TwinID, d.ContractID, name)
			return &WorkloadWithID{
				Workload: wl,
				ID:       id,
			}, nil
		}
	}

	return nil, errors.Wrapf(ErrWorkloadNotFound, "no workload with name '%s'", name)
}

// GetType gets a reservation by name if only of the correct type.
func (d *Deployment) GetType(name Name, typ WorkloadType) (*WorkloadWithID, error) {
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
			id, err := NewWorkloadID(d.TwinID, d.ContractID, wl.Name)
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
// for the steps that the node needs to take to move from d to n. unchanged workloads results
// will be set on n as is
func (d *Deployment) Upgrade(n *Deployment) (*Upgrade, error) {
	if err := n.Valid(); err != nil {
		return nil, errors.Wrap(err, "new deployment is invalid")
	}

	if d.TwinID != n.TwinID || d.ContractID != n.ContractID {
		return nil, fmt.Errorf("cannot change deployment or twin id")
	}

	expected := d.Version + 1
	if expected != n.Version {
		return nil, fmt.Errorf("expecting deployment version %d, got %d", expected, n.Version)
	}

	current := make(map[Name]*Workload)
	for i := range d.Workloads {
		wl := &d.Workloads[i]

		current[wl.Name] = wl
	}

	update := make([]*WorkloadWithID, 0)
	add := make([]*WorkloadWithID, 0)
	remove := make([]*WorkloadWithID, 0)

	for i := range n.Workloads {
		l := &n.Workloads[i]
		id, err := NewWorkloadID(n.TwinID, n.ContractID, l.Name)
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
			// update the state of new deployment
			wl.Result = older.Result
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
		id, _ := NewWorkloadID(d.TwinID, d.ContractID, wl.Name)

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
