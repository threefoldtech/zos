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

func (b *boltStorage) u64(u uint64) []byte {
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], u)
	return v[:]
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

func (b *boltStorage) Get(twin uint32, deployment uint64) (gridtypes.Deployment, error) {
	panic("unimplemented")
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
	panic("unimplemented")
}

func (b *boltStorage) ByTwin(twin uint32) ([]uint64, error) {
	panic("unimplemented")
}
