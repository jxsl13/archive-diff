# archive-diff

You can diff the contents of two folders, archives, rpm packages or any of the previous combinations.

## Installation

```shell
go install github.com/jxsl13/archive-diff@latest
```

Bleeding edge:
```shell
go install github.com/jxsl13/archive-diff@main
```

## Usage

```text
$ archive-diff --help

  DIFF_DIRS_ONLY     only compare directories (default: "false")
  DIFF_FILES_ONLY    only compare files or symlinks (default: "false")
  DIFF_PERM_ONLY     only compare file permissions and sticky bit (default: "false")
  DIFF_OWNER_ONLY    only compare owner, group, gid and uid (default: "false")
  DIFF_EXCLUDE       exclude file paths matching regular expression after cut operation (default: "^$")
  DIFF_INCLUDE       include file paths matching regular expression after cut operation (default: ".*")
  DIFF_CUT           cut ^prefix or suffix$ or any other regular expression before comparing archive paths (default: "^$")

Usage:
  archive-diff a.tar.gz b.tar.xz [flags]
  archive-diff [command]

Available Commands:
  completion  Generate completion script
  help        Help about any command

Flags:
  -c, --cut string       cut ^prefix or suffix$ or any other regular expression before comparing archive paths (default "^$")
  -d, --dirs-only        only compare directories
  -e, --exclude string   exclude file paths matching regular expression after cut operation (default "^$")
  -f, --files-only       only compare files or symlinks
  -h, --help             help for archive-diff
  -i, --include string   include file paths matching regular expression after cut operation (default ".*")
  -o, --owner-only       only compare owner, group, gid and uid
  -p, --perm-only        only compare file permissions and sticky bit

Use "archive-diff [command] --help" for more information about a command.
```

Example usage:
```shell
archive-diff -d whatever-1.0.0-1.noarch.rpm whatever.tar.gz > archive.diff
```
