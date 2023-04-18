package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jxsl13/archive-diff/archive"
)

var (
	archives []string
	option   = ""
)

func init() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("usage: archive-diff [-d|-f] a.tar.gz b.tar.gz")
		os.Exit(1)
	}

	switch len(os.Args) {
	case 3:
		archives = os.Args[1:3]
	case 4:
		option = os.Args[1]
		archives = os.Args[2:4]
	}

	switch option {
	case "-d", "--directory", "--directories":
		option = "d"
	case "-f", "--file", "--files":
		option = "f"
	case "":

	default:
		log.Fatalf("unknown option %q: try --directory or --file\n", option)
	}

	for idx, a := range archives {
		if !archive.IsSupported(a) {
			fmt.Printf("unsupported archive format(%s): %s\n", filepath.Ext(a), a)
			os.Exit(1)
		}
		abs, err := filepath.Abs(a)
		if err != nil {
			log.Fatalln(err)
		}
		archives[idx] = abs
	}
}

func main() {
	source, target := archives[0], archives[1]
	sourceMap, targetMap := make(map[string]File, 1024), make(map[string]File, 1024)
	checkErr(readArchive(option, source, sourceMap))
	checkErr(readArchive(option, target, targetMap))

	added, removed, unchanged, changed := diff(sourceMap, targetMap)

	if len(changed) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- changed files (%s -> %s)---\n", source, target)
		for _, k := range sortedKeys(changed) {
			d := changed[k]
			fmt.Printf("changed: %-"+strconv.Itoa(max+1)+"s %s -> %s (%s uid=%d gid=%d -> %s uid=%d gid=%d)\n",
				k,
				d.Source.Perm(),
				d.Target.Perm(),
				d.Source.Mode,
				d.Source.Uid,
				d.Source.Gid,
				d.Target.Mode,
				d.Target.Uid,
				d.Target.Gid,
			)
		}
	}

	if len(added) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- added files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(added) {
			d := added[k]
			fmt.Printf("added: %-"+strconv.Itoa(max+1)+"s %s (%s)\n", d.Path, d.Perm(), d.Mode)
		}
	}

	if len(removed) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- removed files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(removed) {
			d := removed[k]
			fmt.Printf("removed: %-"+strconv.Itoa(max+1)+"s %s (%s)\n", d.Path, d.Perm(), d.Mode)
		}
	}

	if len(unchanged) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- unchanged files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(unchanged) {
			d := unchanged[k]
			fmt.Printf("unachanged: %-"+strconv.Itoa(max+1)+"s %s (%s)\n", d.Path, d.Perm(), d.Mode)
		}
	}

}

func readArchive(option string, root string, out map[string]File) error {
	return archive.Walk(root, func(path string, info fs.FileInfo, _ io.ReaderAt, err error) error {
		if err != nil {
			return fmt.Errorf("failed to process file: %s: %w", path, err)
		}

		switch option {
		case "f":
			if info.IsDir() {
				return nil
			}
		case "d":
			if !info.IsDir() {
				return nil
			}
		}

		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, root)
		path = strings.TrimPrefix(path, "/")

		out[path] = File{
			Path: path,
			Mode: info.Mode(),
			Uid:  UserId(info),
			Gid:  GroupId(info),
		}
		return nil
	})
}

func diff(source, target map[string]File) (added, removed, unchanged map[string]File, changed map[string]Diff) {
	added, removed, unchanged = make(map[string]File, 64), make(map[string]File, 64), make(map[string]File, 64)
	changed = make(map[string]Diff, 64)

	for t, tf := range target {
		sf, found := source[t]
		if !found {
			added[t] = tf
		} else if sf != tf {
			// found && not equal
			changed[t] = Diff{
				Source: sf,
				Target: tf,
			}
		} else {
			// found && equal
			unchanged[t] = tf
		}
	}

	for s, sf := range source {
		_, found := target[s]
		if !found {
			removed[s] = sf
		}
	}

	return added, removed, unchanged, changed
}

func sortedKeys[V any](m map[string]V) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}

	sort.Strings(result)
	return result
}

func longestKey[V any](m map[string]V) int {
	max := 0
	for k := range m {
		l := len([]rune(k))
		if l > max {
			max = l
		}
	}

	return max
}
