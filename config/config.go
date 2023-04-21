package config

import (
	"fmt"
	"regexp"

	"github.com/jxsl13/archive-diff/model"
)

type Config struct {
	DirsOnly  bool   `koanf:"dirs.only" short:"d" description:"only compare directories"`
	FilesOnly bool   `koanf:"files.only" short:"f" description:"only compare files or symlinks"`
	PermOnly  bool   `koanf:"perm.only" short:"p" description:"only compare file permissions and sticky bit"`
	OwnerOnly bool   `koanf:"owner.only" short:"o" description:"only compare owner, group, gid and uid"`
	Exclude   string `koanf:"exclude" short:"e" description:"exclude file paths matching regular expression"`
	Include   string `koanf:"include" short:"i" description:"include file paths matching regular expression"`

	FileOption   string
	Equal        func(a, b model.File) bool
	ExcludeRegex *regexp.Regexp
	IncludeRegex *regexp.Regexp
}

func (c *Config) Validate() error {
	if c.DirsOnly && c.FilesOnly {
		return fmt.Errorf("may only define -d or -f, not both")
	} else if c.DirsOnly {
		c.FileOption = "d"
	} else if c.FilesOnly {
		c.FileOption = "f"
	}

	if c.PermOnly && c.OwnerOnly {
		return fmt.Errorf("may only define -p or -o, not both")
	} else if c.PermOnly {
		c.Equal = func(a, b model.File) bool {
			return a.Perm() == b.Perm()
		}
	} else if c.OwnerOnly {
		c.Equal = func(a, b model.File) bool {
			return a.Owner == b.Owner
		}
	} else {
		c.Equal = func(a, b model.File) bool {
			return a == b
		}
	}

	r, err := regexp.Compile(c.Exclude)
	if err != nil {
		return fmt.Errorf("invalid exclude regex: %w", err)
	}
	c.ExcludeRegex = r

	r, err = regexp.Compile(c.Include)
	if err != nil {
		return fmt.Errorf("invalid include regex: %w", err)
	}
	c.IncludeRegex = r

	return nil
}
