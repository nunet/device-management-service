package types

type AllocationKeyType string

const (
	KeySSH AllocationKeyType = "ssh"
	KeyGPG AllocationKeyType = "gpg"
)

// AllocationKey is a key specification to be uploaded on the allocation, e.g. ssh, gpg
type AllocationKey struct {
	Type AllocationKeyType `json:"type"`
	File string            `json:"file"` // source path to file
	Dest string            `json:"dest"` // destination path
}

func (t AllocationKeyType) Equal(other string) bool {
	return string(t) == other
}

func (t AllocationKeyType) String() string {
	return string(t)
}
