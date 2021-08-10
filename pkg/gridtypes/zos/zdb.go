package zos

import (
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// ZDBMode is the enumeration of the modes 0-db can operate in
type ZDBMode string

// Enumeration of the modes 0-db can operate in
const (
	ZDBModeUser = "user"
	ZDBModeSeq  = "seq"
)

func (m ZDBMode) String() string {
	return string(m)
}

// Valid validates the mode
func (m ZDBMode) Valid() error {
	if m != ZDBModeUser && m != ZDBModeSeq {
		return fmt.Errorf("invalid zdb mode")
	}

	return nil
}

// ZDB namespace creation info
type ZDB struct {
	Size     gridtypes.Unit `json:"size"`
	Mode     ZDBMode        `json:"mode"`
	Password string         `json:"password"`
	Public   bool           `json:"public"`
}

//Valid implementation
func (z ZDB) Valid(getter gridtypes.WorkloadGetter) error {
	if z.Size == 0 {
		return fmt.Errorf("invalid size")
	}

	if err := z.Mode.Valid(); err != nil {
		return fmt.Errorf("invalid mode")
	}

	return nil
}

// Challenge implementation
func (z ZDB) Challenge(b io.Writer) error {

	if _, err := fmt.Fprintf(b, "%d", z.Size); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", z.Mode.String()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", z.Password); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%t", z.Public); err != nil {
		return err
	}

	return nil
}

// Capacity implements WorkloadData
func (z ZDB) Capacity() (cap gridtypes.Capacity, err error) {
	cap.HRU = z.Size
	return
}

// ZDBResult is the information return to the BCDB
// after deploying a 0-db namespace
type ZDBResult struct {
	Namespace string
	IPs       []string
	Port      uint
}
