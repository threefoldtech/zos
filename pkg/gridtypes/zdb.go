package gridtypes

// ZDBMode is the enumeration of the modes 0-db can operate in
type ZDBMode string

// Enumeration of the modes 0-db can operate in
const (
	ZDBModeUser = "user"
	ZDBModeSeq  = "seq"
)

// ZDB namespace creation info
type ZDB struct {
	Size     uint64     `json:"size"`
	Mode     ZDBMode    `json:"mode"`
	Password string     `json:"password"`
	DiskType DeviceType `json:"disk_type"`
	Public   bool       `json:"public"`

	PlainPassword string `json:"-"`
}

// ZDBResult is the information return to the BCDB
// after deploying a 0-db namespace
type ZDBResult struct {
	Namespace string
	IPs       []string
	Port      uint
}
