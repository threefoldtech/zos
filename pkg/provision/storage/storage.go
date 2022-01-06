package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

var (
	ErrTransactionNotExist = fmt.Errorf("no transaction found")
	ErrInvalidWorkloadType = fmt.Errorf("invalid workload type")
)

const (
	keyVersion              = "version"
	keyMetadata             = "metadata"
	keyDescription          = "description"
	keySignatureRequirement = "signature_requirement"
	keyWorkloads            = "workloads"
	keyTransactions         = "transactions"
)

type boltStorage struct {
	db *bolt.DB
}

func New(path string) (provision.Storage, error) {
	db, err := bolt.Open(path, 0644, bolt.DefaultOptions)
	if err != nil {
		return nil, err
	}

	return &boltStorage{
		db,
	}, nil
}

func (b *boltStorage) u32(u uint32) []byte {
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], u)
	return v[:]
}

func (b *boltStorage) l32(v []byte) uint32 {
	return binary.BigEndian.Uint32(v)
}

func (b *boltStorage) u64(u uint64) []byte {
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], u)
	return v[:]
}

func (b *boltStorage) l64(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}

func (b *boltStorage) Create(deployment *gridtypes.Deployment) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		twin, err := tx.CreateBucketIfNotExists(b.u32(deployment.TwinID))
		if err != nil {
			return errors.Wrap(err, "failed to create twin")
		}
		dl, err := twin.CreateBucket(b.u64(deployment.ContractID))
		if errors.Is(err, bolt.ErrBucketExists) {
			return provision.ErrDeploymentExists
		} else if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		if err := dl.Put([]byte(keyVersion), b.u32(deployment.Version)); err != nil {
			return err
		}
		if err := dl.Put([]byte(keyDescription), []byte(deployment.Description)); err != nil {
			return err
		}
		if err := dl.Put([]byte(keyMetadata), []byte(deployment.Metadata)); err != nil {
			return err
		}
		sig, err := json.Marshal(deployment.SignatureRequirement)
		if err != nil {
			return errors.Wrap(err, "failed to encode signature requirement")
		}
		if err := dl.Put([]byte(sig), sig); err != nil {
			return err
		}
		return nil
	})
}

func (b *boltStorage) Delete(twin uint32, deployment uint64) error {
	panic("unimplemented")
}

func (b *boltStorage) Get(twin uint32, deployment uint64) (dl gridtypes.Deployment, err error) {
	dl.TwinID = twin
	dl.ContractID = deployment
	err = b.db.View(func(t *bolt.Tx) error {
		twin := t.Bucket(b.u32(twin))
		if twin == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "twin not found")
		}
		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "deployment not found")
		}
		if value := deployment.Get([]byte(keyVersion)); value != nil {
			dl.Version = b.l32(value)
		}
		if value := deployment.Get([]byte(keyDescription)); value != nil {
			dl.Description = string(value)
		}
		if value := deployment.Get([]byte(keyMetadata)); value != nil {
			dl.Metadata = string(value)
		}
		if value := deployment.Get([]byte(keySignatureRequirement)); value != nil {
			if err := json.Unmarshal(value, &dl.SignatureRequirement); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return dl, err
	}

	dl.Workloads, err = b.workloads(twin, deployment)
	return
}

func (b *boltStorage) Error(twin uint32, deployment uint64, err error) error {
	panic("unimplemented")
}

func (b *boltStorage) Add(twin uint32, deployment uint64, name gridtypes.Name, typ gridtypes.WorkloadType, global bool) error {
	if global {
		panic("TODO: not implemented")
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		twin := tx.Bucket(b.u32(twin))
		if twin == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "twin not found")
		}
		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "deployment not found")
		}

		workloads, err := deployment.CreateBucketIfNotExists([]byte(keyWorkloads))
		if err != nil {
			return errors.Wrap(err, "failed to prepare workloads storage")
		}

		if value := workloads.Get([]byte(name)); value != nil {
			return errors.Wrap(provision.ErrWorkloadExists, "workload with same name already exists in deployment")
		}

		return workloads.Put([]byte(name), []byte(typ.String()))
	})
}

func (b *boltStorage) Remove(twin uint32, deployment uint64, name gridtypes.Name) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		twin := tx.Bucket(b.u32(twin))
		if twin == nil {
			return nil
		}

		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return nil
		}

		workloads := deployment.Bucket([]byte(keyWorkloads))
		if workloads == nil {
			return nil
		}

		return workloads.Delete([]byte(name))
	})
}

func (b *boltStorage) Transaction(twin uint32, deployment uint64, workload gridtypes.Workload) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		if err := workload.Result.Valid(); err != nil {
			return errors.Wrap(err, "failed to validate workload result")
		}

		data, err := json.Marshal(workload)
		if err != nil {
			return errors.Wrap(err, "failed to encode workload data")
		}

		twin := tx.Bucket(b.u32(twin))
		if twin == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "twin not found")
		}
		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "deployment not found")
		}

		workloads := deployment.Bucket([]byte(keyWorkloads))
		if workloads == nil {
			return errors.Wrap(provision.ErrWorkloadNotExist, "deployment has no active workloads")
		}

		typRaw := workloads.Get([]byte(workload.Name))
		if typRaw == nil {
			return errors.Wrap(provision.ErrWorkloadNotExist, "workload does not exist")
		}

		if workload.Type != gridtypes.WorkloadType(typRaw) {
			return errors.Wrapf(ErrInvalidWorkloadType, "invalid workload type, expecting '%s'", string(typRaw))
		}

		logs, err := deployment.CreateBucketIfNotExists([]byte(keyTransactions))
		if err != nil {
			return errors.Wrap(err, "failed to prepare deployment transaction logs")
		}

		id, err := logs.NextSequence()
		if err != nil {
			return err
		}

		return logs.Put(b.u64(id), data)
	})
}

func (b *boltStorage) workloads(twin uint32, deployment uint64) ([]gridtypes.Workload, error) {
	names := make(map[gridtypes.Name]gridtypes.WorkloadType)
	workloads := make(map[gridtypes.Name]gridtypes.Workload)

	err := b.db.View(func(tx *bolt.Tx) error {
		twin := tx.Bucket(b.u32(twin))
		if twin == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "twin not found")
		}
		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "deployment not found")
		}

		types := deployment.Bucket([]byte(keyWorkloads))
		if types == nil {
			// no active workloads
			return nil
		}

		types.ForEach(func(k, v []byte) error {
			names[gridtypes.Name(k)] = gridtypes.WorkloadType(v)
			return nil
		})

		if len(names) == 0 {
			return nil
		}

		logs := deployment.Bucket([]byte(keyTransactions))
		if logs == nil {
			// should we return an error instead?
			return nil
		}

		cursor := logs.Cursor()

		for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
			var workload gridtypes.Workload
			if err := json.Unmarshal(v, &workload); err != nil {
				return errors.Wrap(err, "error while scanning transcation logs")
			}

			if _, ok := workloads[workload.Name]; ok {
				// already loaded and have last state
				continue
			}

			typ, ok := names[workload.Name]
			if !ok {
				// not an active workload
				continue
			}

			if workload.Type != typ {
				return fmt.Errorf("database inconsistency wrong workload type")
			}

			// otherwise we have a match.
			if workload.Result.State == gridtypes.StateUnChanged {
				continue
			}

			workloads[workload.Name] = workload
			if len(workloads) == len(names) {
				// we all latest states of active workloads
				break
			}
		}

		return nil
	})

	for name, typ := range names {
		if _, ok := workloads[name]; ok {
			continue
		}

		// otherwise we need to put a place holder here. this
		// can happen if a workload was added to the deployment
		// but no transactions has been registered for this workload
		// yet.
		workloads[name] = gridtypes.Workload{
			Name: name,
			Type: typ,
			Result: gridtypes.Result{
				State: gridtypes.StateScheduled,
			},
		}
	}

	result := make([]gridtypes.Workload, 0, len(workloads))

	for _, wl := range workloads {
		result = append(result, wl)
	}

	return result, err
}

func (b *boltStorage) Current(twin uint32, deployment uint64, name gridtypes.Name) (gridtypes.Workload, error) {
	var workload gridtypes.Workload
	err := b.db.View(func(tx *bolt.Tx) error {
		twin := tx.Bucket(b.u32(twin))
		if twin == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "twin not found")
		}
		deployment := twin.Bucket(b.u64(deployment))
		if deployment == nil {
			return errors.Wrap(provision.ErrDeploymentNotExists, "deployment not found")
		}

		workloads := deployment.Bucket([]byte(keyWorkloads))
		if workloads == nil {
			return errors.Wrap(provision.ErrWorkloadNotExist, "deployment has no active workloads")
		}

		// this checks if this workload is an "active" workload.
		// if workload is not in this map, then workload might have been
		// deleted.
		typRaw := workloads.Get([]byte(name))
		if typRaw == nil {
			return errors.Wrap(provision.ErrWorkloadNotExist, "workload does not exist")
		}

		typ := gridtypes.WorkloadType(typRaw)

		logs := deployment.Bucket([]byte(keyTransactions))
		if logs == nil {
			return errors.Wrap(ErrTransactionNotExist, "no transaction logs available")
		}

		cursor := logs.Cursor()

		found := false
		for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
			if err := json.Unmarshal(v, &workload); err != nil {
				return errors.Wrap(err, "error while scanning transcation logs")
			}

			if workload.Name != name {
				continue
			}

			if workload.Type != typ {
				return fmt.Errorf("database inconsistency wrong workload type")
			}

			// otherwise we have a match.
			if workload.Result.State == gridtypes.StateUnChanged {
				continue
			}
			found = true
			break
		}

		if !found {
			return ErrTransactionNotExist
		}

		return nil
	})

	return workload, err
}

func (b *boltStorage) Twins() ([]uint32, error) {
	var twins []uint32
	err := b.db.View(func(t *bolt.Tx) error {
		curser := t.Cursor()
		for k, v := curser.First(); k != nil; k, v = curser.Next() {
			if v != nil {
				// checking that it is a bucket
				continue
			}

			if len(k) != 4 {
				// sanity check it's a valid uint32
				continue
			}

			twins = append(twins, b.l32(k))
		}

		return nil
	})

	return twins, err
}

func (b *boltStorage) ByTwin(twin uint32) ([]uint64, error) {
	var deployments []uint64
	err := b.db.View(func(t *bolt.Tx) error {
		bucket := t.Bucket(b.u32(twin))
		if bucket == nil {
			return nil
		}

		curser := bucket.Cursor()
		for k, v := curser.First(); k != nil; k, v = curser.Next() {
			if v != nil {
				// checking that it is a bucket
				continue
			}

			if len(k) != 8 {
				// sanity check it's a valid uint32
				continue
			}

			deployments = append(deployments, b.l64(k))
		}

		return nil
	})

	return deployments, err
}
