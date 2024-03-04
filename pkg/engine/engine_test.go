package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type Disk struct {
	Path string
	Size uint64
}

type VM struct {
	disks []string
}

type DiskManager struct {
	BaseResource[Disk]
}

type VmManager struct {
	BaseResource[VM]
}

func (m *DiskManager) create(ctx Context, size uint64) (void Void, err error) {
	fmt.Println("create a disk")
	path := fmt.Sprintf("/dev/%d.disk", size)
	disk := Disk{
		Path: path,
		Size: size,
	}

	return void, m.Set(ctx, disk)
}

func (m *DiskManager) delete(ctx Context, _ Void) (void Void, err error) {
	// actually delete current object
	return
}

func (m *VmManager) create(ctx Context, _ Void) (void Void, err error) {
	return void, m.Set(ctx, VM{})
}

func (m *VmManager) addDisk(ctx Context, disk string) (void Void, err error) {

	vm, err := m.Current(ctx)
	if err != nil {
		return void, err
	}

	err = ctx.Use(func(ctx Context, r []Record) error {
		// add the actual disk from the record
		// assume success
		vm.disks = append(vm.disks, disk)
		return m.Set(ctx, vm)
	}, disk)

	return
}

func (m *VmManager) deleteDisk(ctx Context, disk string) (void Void, err error) {
	vm, err := m.Current(ctx)
	if err != nil {
		return void, err
	}

	err = ctx.UnUse(func(ctx Context, r []Record) error {
		// add the actual disk from the record
		// assume success
		vm.disks = slices.DeleteFunc(vm.disks, func(d string) bool {
			return disk == d
		})

		return m.Set(ctx, vm)
	}, disk)

	return
}

func TestEngine(t *testing.T) {
	mem := NewMemStore()
	engine := NewEngine(mem)

	var disk DiskManager
	var vm VmManager

	engine.
		Resource(
			// exclusive resource means the disk can't be used
			// by enslaved by more than one resource
			NewResourceBuilder[Disk](ResourceExclusive).
				WithAction("create",
					NewAction(disk.create),
					ServiceObjectMustNotExist).
				WithDelete(NewAction(disk.delete)).
				Build(),
		).
		Resource(
			NewResourceBuilder[VM]().
				WithAction("create",
					NewAction(vm.create),
					ServiceObjectMustNotExist).
				WithAction("add-disk",
					NewAction(vm.addDisk)).
				WithAction("del-disk",
					NewAction(vm.deleteDisk)).
				Build())
	require.NotNil(t, engine)

	// create space in storage
	mem.SpaceCreate(0, "default")

	// adding a disk to a vm that does not exist
	// should return an error
	_, err := engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "add-disk",
			ResourceID: "vm0",
			Payload:    []byte(`"disk0"`),
		},
	})

	require.ErrorIs(t, err, ErrObjectDoesNotExist)

	// let's now try proper cases

	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "create",
			ResourceID: "vm0",
			Payload:    []byte(`null`),
		},
	})

	require.NoError(t, err)

	space, _ := mem.getSpace(0, "default")

	vm0 := space.objects["vm0"]
	require.EqualValues(t, "vm0", vm0.ID)
	require.EqualValues(t, "VM", vm0.Type)
	require.Empty(t, vm0.Masters)
	require.EqualValues(t, "{}", vm0.Data)

	// add a disk that does not exist

	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "add-disk",
			ResourceID: "vm0",
			Payload:    []byte(`"disk0"`),
		},
	})

	// because disk does not exist
	require.ErrorIs(t, err, ErrObjectDoesNotExist)

	// create the disk first
	_, err = engine.Handle(context.Background(), Request{
		Type:  "Disk",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "create",
			ResourceID: "disk0",
			Payload:    []byte(`500`), // size
		},
	})

	require.NoError(t, err)

	// add the disk to the vm
	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "add-disk",
			ResourceID: "vm0",
			Payload:    []byte(`"disk0"`),
		},
	})

	// because disk does not exist
	require.NoError(t, err)

	// now we can check memory again
	require.Len(t, space.objects, 2)

	disk0 := space.objects["disk0"]
	require.EqualValues(t, "disk0", disk0.ID)
	require.EqualValues(t, "Disk", disk0.Type)
	require.EqualValues(t, []string{"vm0"}, disk0.Masters)

	var loaded Disk
	require.NoError(t, json.Unmarshal(disk0.Data, &loaded))

	require.EqualValues(t, loaded, Disk{Size: 500, Path: "/dev/500.disk"})

	// test exclusivity of the disk
	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "create",
			ResourceID: "vm1",
			Payload:    []byte(`null`),
		},
	})

	require.NoError(t, err)

	// add the disk to the vm
	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "add-disk",
			ResourceID: "vm1",
			Payload:    []byte(`"disk0"`),
		},
	})

	// because disk does not exist
	require.ErrorIs(t, err, ErrObjectInUse)

	// we also can't delete disk0 because it's been used

	_, err = engine.Handle(context.Background(), Request{
		Type:  "Disk",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "delete",
			ResourceID: "disk0",
			Payload:    []byte(`null`),
		},
	})

	require.ErrorIs(t, err, ErrObjectInUse)

	// but now we can delete the disk from the vm first

	// delete the disk from the vm
	_, err = engine.Handle(context.Background(), Request{
		Type:  "VM",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "del-disk",
			ResourceID: "vm0",
			Payload:    []byte(`"disk0"`),
		},
	})

	require.NoError(t, err)

	// then delete the disk

	_, err = engine.Handle(context.Background(), Request{
		Type:  "Disk",
		User:  0,
		Space: "default",
		ResourceRequest: ResourceRequest{
			Action:     "delete",
			ResourceID: "disk0",
			Payload:    []byte(`null`),
		},
	})

	require.NoError(t, err)

	// only 2 vms remaining
	require.Len(t, space.objects, 2)
}
