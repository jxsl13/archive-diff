package main

import (
	"fmt"
	"io/fs"
)

type File struct {
	Path string
	Mode fs.FileMode

	Username  string
	Groupname string
	Uid       int
	Gid       int
}

func (f File) Owner() string {
	return fmt.Sprintf("%s:%s (%d:%d)", f.Username, f.Groupname, f.Uid, f.Gid)
}

func (f File) Perm() string {
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

type Diff struct {
	Source File
	Target File
}
