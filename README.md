# archive-diff

You can diff the contents of two folders, archives, rpm packages or and of the previous combinations.

## Usage

```shell
archive-diff [-d|-f] a.tar.gz b.tar.gz

-d is an optional parameter that shows only a diff between the directories in the archive.
-f does only show a diff between the files and symlinks in the directory
```

Example usage:
```shell
archive-diff -d whatever-1.0.0-1.noarch.rpm whatever.tar.gz > archive.diff
```

### Installation

```
go install github.com/jxsl13/archive-diff@latest
```