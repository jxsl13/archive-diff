package model

import (
	"fmt"
	"io/fs"
	"os"
)

type File struct {
	Path string
	Mode fs.FileMode
	Owner
}

var ownerFormat = "%s:%s (%d:%d)"

// SetOwnerFormat allows to increase the format spacing by providing the corresponding parameters.
func SetOwnerFormat(maxUser, maxGroup, maxUid, maxGid int) {
	ownerFormat = fmt.Sprintf("%%%ds:%%-%ds (%%%dd:%%-%dd)", maxUser, maxGroup, maxUid, maxGid)
}

func (f File) OwnerString() string {
	return fmt.Sprintf(ownerFormat, f.Username, f.Groupname, f.Uid, f.Gid)
}

func (f File) Perm() os.FileMode {
	const permMask = os.ModeSticky | os.ModePerm
	return f.Mode & permMask
}

func (f File) PermString() string {
	mode := f.Mode
	sticky := "0"
	if mode&fs.ModeSticky != 0 {
		sticky = "1"
	}
	return fmt.Sprintf("%s%o",
		sticky,
		mode.Perm(),
	)
}
