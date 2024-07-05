package filesystem

import "context"

type bcachefsUtils struct {
	executer
}

func newBcachefsCmd(exec executer) bcachefsUtils {
	return bcachefsUtils{exec}
}

// SubvolumeAdd adds a new subvolume at path
func (u *bcachefsUtils) SubvolumeAdd(ctx context.Context, root string) error {
	_, err := u.run(ctx, "bcachefs", "subvolume", "create", root)
	return err
}

// SubvolumeRemove removes a subvolume
func (u *bcachefsUtils) SubvolumeRemove(ctx context.Context, root string) error {
	_, err := u.run(ctx, "bcachefs", "subvolume", "delete", root)
	return err
}
