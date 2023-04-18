package main

import (
	"archive/tar"
	"log"
	"os"
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

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
