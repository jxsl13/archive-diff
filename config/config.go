package config

import (
	"fmt"
	"regexp"
)

type Config struct {
	DirsOnly  bool   `koanf:"dirs.only" short:"d" description:"only compare directories"`
	FilesOnly bool   `koanf:"files.only" short:"f" description:"only compare files or symlinks"`
	Exclude   string `koanf:"exclude" short:"e" description:"exclude file paths matching regular expression"`
	Include   string `koanf:"include" short:"i" description:"include file paths matching regular expression"`

	Option       string
	ExcludeRegex *regexp.Regexp
	IncludeRegex *regexp.Regexp
}

func (c *Config) Validate() error {
	if c.DirsOnly && c.FilesOnly {
		return fmt.Errorf("may only define -d or -f, not both")
	} else if c.DirsOnly {
		c.Option = "d"
	} else if c.FilesOnly {
		c.Option = "f"
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
