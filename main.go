package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jxsl13/archive-diff/archive"
	"github.com/jxsl13/archive-diff/config"
	"github.com/jxsl13/archive-diff/model"
	"github.com/spf13/cobra"
)

func main() {
	NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	rootContext := rootContext{}

	// rootCmd represents the run command
	rootCmd := &cobra.Command{
		Use:   "archive-diff a.tar.gz b.tar.xz",
		Short: "diff two archives, folders or a rpm packages and any of the previous",
		Args:  cobra.ExactArgs(2),
		RunE:  rootContext.RunE,
	}

	// register flags but defer parsing and validation of the final values
	rootCmd.PreRunE = rootContext.PreRunE(rootCmd)

	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: fmt.Sprintf(`To load completions:

	Bash:

	  $ source <(%[1]s completion bash)

	  # To load completions for each session, execute once:
	  # Linux:
	  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
	  # macOS:
	  $ %[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s

	Zsh:

	  # If shell completion is not already enabled in your environment,
	  # you will need to enable it.  You can execute the following once:

	  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

	  # To load completions for each session, execute once:
	  $ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

	  # You will need to start a new shell for this setup to take effect.

	fish:

	  $ %[1]s completion fish | source

	  # To load completions for each session, execute once:
	  $ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

	PowerShell:

	  PS> %[1]s completion powershell | Out-String | Invoke-Expression

	  # To load completions for every new session, run:
	  PS> %[1]s completion powershell > %[1]s.ps1
	  # and source this file from your PowerShell profile.
	`, rootCmd.Root().Name()),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	rootCmd.AddCommand(completionCmd)

	return rootCmd
}

type rootContext struct {
	Config     *config.Config
	SourcePath string
	TargetPath string
}

func (c *rootContext) PreRunE(cmd *cobra.Command) func(cmd *cobra.Command, args []string) error {
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

func (c *rootContext) RunE(cmd *cobra.Command, args []string) (err error) {
	sourceMap, targetMap := make(map[string]model.File, 1024), make(map[string]model.File, 1024)
	source, target := c.SourcePath, c.TargetPath
	include, exclude := c.Config.IncludeRegex, c.Config.ExcludeRegex

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		checkErr(readArchive(c.Config.FileOption, source, include, exclude, sourceMap))
	}()

	go func() {
		defer wg.Done()
		checkErr(readArchive(c.Config.FileOption, target, include, exclude, targetMap))
	}()

	wg.Wait()

	added, removed, unchanged, changed, u, g, ui, gi := diff(c.Config.Equal, sourceMap, targetMap)
	model.SetOwnerFormat(len(u), len(g), len(ui), len(gi))

	if len(changed) > 0 {
		max := longestKey(changed)
		fmt.Printf("--- changed files (%s -> %s)---\n", source, target)
		for _, k := range sortedKeys(changed) {
			d := changed[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s -> %s %12s %s\n",
				k,
				d.Source.PermString(),
				d.Source.Mode,
				d.Source.OwnerString(),
				d.Target.PermString(),
				d.Target.Mode,
				d.Target.OwnerString(),
			)
		}
	}

	if len(added) > 0 {
		max := longestKey(added)
		fmt.Printf("--- added files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(added) {
			d := added[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.PermString(), d.Mode, d.OwnerString())
		}
	}

	if len(removed) > 0 {
		max := longestKey(removed)
		fmt.Printf("--- removed files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(removed) {
			d := removed[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.PermString(), d.Mode, d.OwnerString())
		}
	}

	if len(unchanged) > 0 {
		max := longestKey(unchanged)
		fmt.Printf("--- unchanged files (%s -> %s) ---\n", source, target)
		for _, k := range sortedKeys(unchanged) {
			d := unchanged[k]
			fmt.Printf("%-"+strconv.Itoa(max+1)+"s %s %12s %s\n", d.Path, d.PermString(), d.Mode, d.OwnerString())
		}
	}

	return nil
}

func readArchive(fileOption string, root string, include, exclude *regexp.Regexp, out map[string]model.File) error {
	return archive.Walk(root, func(path string, info fs.FileInfo, _ io.ReaderAt, err error) error {
		if err != nil {
			return fmt.Errorf("failed to process file: %s: %w", path, err)
		}

		switch fileOption {
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

		out[path] = model.File{
			Path: path,
			Mode: info.Mode(),
			Owner: model.Owner{
				Username:  Username(info),
				Groupname: Groupname(info),
				Uid:       UserId(info),
				Gid:       GroupId(info),
			},
		}
		return nil
	})
}

func diff(equal func(a, b model.File) bool, source, target map[string]model.File) (
	added map[string]model.File,
	removed map[string]model.File,
	unchanged map[string]model.File,
	changed map[string]model.Diff,
	longestUser string,
	longestGroup string,
	longestUid string,
	longestGid string,
) {
	added, removed, unchanged = make(map[string]model.File, 64), make(map[string]model.File, 64), make(map[string]model.File, 64)
	changed = make(map[string]model.Diff, 64)

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
		} else if !equal(sf, tf) {
			// found && not equal
			changed[t] = model.Diff{
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
