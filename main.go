package main

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jxsl13/archive-diff/archive"
	"github.com/jxsl13/archive-diff/config"
	"github.com/spf13/cobra"
)

func main() {
	NewRunCmd().Execute()
}

func NewRunCmd() *cobra.Command {
	runContext := runContext{}

	// runCmd represents the run command
	runCmd := &cobra.Command{
		Use:   "archive-diff a.tar.gz b.tar.xz",
		Short: "diff two archives, folders or a rpm packages and any of the previous",
		Args:  cobra.ExactArgs(2),
		RunE:  runContext.RunE,
	}

	// register flags but defer parsing and validation of the final values
	runCmd.PreRunE = runContext.PreRunE(runCmd)

	return runCmd
}

type runContext struct {
	Config     *config.Config
	SourcePath string
	TargetPath string
}

func (c *runContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {
	c.Config = &config.Config{
		DirsOnly:  false,
		FilesOnly: false,
		Exclude:   "^$",
		Include:   ".*",
	}

	runParser := config.RegisterFlags(c.Config, true, cmd)

	return func(cmd *cobra.Command, args []string) error {
		for idx, a := range args {
			if !archive.IsSupported(a) {
				return fmt.Errorf("unsupported archive format(%s): %s", filepath.Ext(a), a)
			}
			abs, err := filepath.Abs(a)
			if err != nil {
				return err
			}
			switch idx {
			case 0:
				c.SourcePath = abs
			case 1:
				c.TargetPath = abs
			}
		}

		return runParser()
	}
}

func (c *runContext) RunE(cmd *cobra.Command, args []string) (err error) {
	sourceMap, targetMap := make(map[string]File, 1024), make(map[string]File, 1024)
	source, target := c.SourcePath, c.TargetPath
	include, exclude := c.Config.IncludeRegex, c.Config.ExcludeRegex

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		checkErr(readArchive(c.Config.Option, source, include, exclude, sourceMap))
	}()

	go func() {
		defer wg.Done()
		checkErr(readArchive(c.Config.Option, target, include, exclude, targetMap))
	}()

	wg.Wait()

	added, removed, unchanged, changed, u, g, ui, gi := diff(sourceMap, targetMap)
	SetOwnerFormat(len(u), len(g), len(ui), len(gi))

	if len(changed) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- changed files (%s -> %s)---\n", source, target)
		for _, k := range sortedKeys(changed) {
			d := changed[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s -> %s %12s %s\n",
				k,
				d.Source.Perm(),
				d.Source.Mode,
				d.Source.Owner(),
				d.Target.Perm(),
				d.Target.Mode,
				d.Target.Owner(),
			)
		}
	}

	if len(added) > 0 {
		max := longestKey(added)
		fmt.Printf("--- added files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(added) {
			d := added[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.Perm(), d.Mode, d.Owner())
		}
	}

	if len(removed) > 0 {
		max := longestKey(removed)
		fmt.Printf("--- removed files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(removed) {
			d := removed[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.Perm(), d.Mode, d.Owner())
		}
	}

	if len(unchanged) > 0 {
		max := longestKey(unchanged)
		fmt.Printf("--- unchanged files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(unchanged) {
			d := unchanged[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.Perm(), d.Mode, d.Owner())
		}
	}

	return nil
}

func readArchive(option string, root string, include, exclude *regexp.Regexp, out map[string]File) error {
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

		if !include.MatchString(path) {
			return nil
		} else if exclude.MatchString(path) {
			// skip
			return nil
		}

		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, root)
		path = strings.TrimPrefix(path, "/")

		out[path] = File{
			Path:      path,
			Mode:      info.Mode(),
			Username:  Username(info),
			Groupname: Groupname(info),
			Uid:       UserId(info),
			Gid:       GroupId(info),
		}
		return nil
	})
}

func diff(source, target map[string]File) (added, removed, unchanged map[string]File, changed map[string]Diff, longestUser, longestGroup, longestUid, longestGid string) {
	added, removed, unchanged = make(map[string]File, 64), make(map[string]File, 64), make(map[string]File, 64)
	changed = make(map[string]Diff, 64)

	var (
		maxUser  []rune
		maxGroup []rune
		maxUid   []rune
		maxGid   []rune
	)

	for t, tf := range target {
		var (
			user  = []rune(tf.Username)
			group = []rune(tf.Groupname)
			uid   = []rune(strconv.FormatUint(uint64(tf.Uid), 10))
			gid   = []rune(strconv.FormatUint(uint64(tf.Gid), 10))
		)
		if len(user) > len(maxUser) {
			maxUser = user
		}
		if len(group) > len(maxGroup) {
			maxGroup = group
		}
		if len(uid) > len(maxUid) {
			maxUid = uid
		}
		if len(gid) > len(maxGid) {
			maxGid = gid
		}

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
		var (
			uid   = []rune(strconv.FormatUint(uint64(sf.Uid), 10))
			gid   = []rune(strconv.FormatUint(uint64(sf.Gid), 10))
			user  = []rune(sf.Username)
			group = []rune(sf.Groupname)
		)
		if len(user) > len(maxUser) {
			maxUser = user
		}
		if len(group) > len(maxGroup) {
			maxGroup = group
		}
		if len(uid) > len(maxUid) {
			maxUid = uid
		}
		if len(gid) > len(maxGid) {
			maxGid = gid
		}
		_, found := target[s]
		if !found {
			removed[s] = sf
		}
	}

	return added, removed, unchanged, changed, string(maxUser), string(maxGroup), string(maxUid), string(maxGid)
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
