package main

import (
	"archive/tar"
	"log"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"

	"github.com/cavaliergopher/cpio"
)

func UserId(fi os.FileInfo) int {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid)
	}

	if stat, ok := fi.Sys().(*tar.Header); ok {
		return int(stat.Uid)
	}

	if stat, ok := fi.Sys().(*cpio.Header); ok {
		return int(stat.Uid)
	}

	return -1
}

var (
	userCache map[uint32]string = make(map[uint32]string, 3)
	umu       sync.Mutex
)

func Username(fi os.FileInfo) string {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		umu.Lock()
		name, found := userCache[stat.Uid]
		umu.Unlock()

		if found {
			return name
		}
		uidStr := strconv.FormatUint(uint64(stat.Uid), 10)
		if user, err := user.LookupId(uidStr); err == nil {
			umu.Lock()
			userCache[stat.Uid] = user.Name
			umu.Unlock()
		}
		return ""
	}

	if stat, ok := fi.Sys().(*tar.Header); ok {
		return stat.Uname
	}

	if _, ok := fi.Sys().(*cpio.Header); ok {
		return ""
	}

	return ""
}

func GroupId(fi os.FileInfo) int {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int(stat.Gid)
	}

	if stat, ok := fi.Sys().(*tar.Header); ok {
		return int(stat.Gid)
	}

	if stat, ok := fi.Sys().(*cpio.Header); ok {
		return int(stat.Guid)
	}

	return -1
}

var (
	groupCache map[uint32]string = make(map[uint32]string, 3)
	gmu        sync.Mutex
)

func Groupname(fi os.FileInfo) string {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		gmu.Lock()
		name, found := groupCache[stat.Gid]
		gmu.Unlock()
		if found {
			return name
		}
		gidStr := strconv.FormatUint(uint64(stat.Gid), 10)
		if user, err := user.LookupId(gidStr); err == nil {
			gmu.Lock()
			userCache[stat.Uid] = user.Name
			gmu.Unlock()
		}
		return ""
	}

	if stat, ok := fi.Sys().(*tar.Header); ok {
		return stat.Gname
	}

	if _, ok := fi.Sys().(*cpio.Header); ok {
		return ""
	}

	return ""
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
