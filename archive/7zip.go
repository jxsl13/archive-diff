package archive

import (
	"io"
	"os"
	"path"

	"github.com/bodgit/sevenzip"
)

func Walk7Zip(file *os.File, fileSize int64, walkFunc WalkFunc) error {
	zfs, err := sevenzip.NewReader(file, fileSize)
	if err != nil {
		return err
	}

	for _, f := range zfs.File {
		err = walk7ZipFile(f, walkFunc)
		if err != nil {
			return err
		}
	}
	return nil
}

func walk7ZipFile(f *sevenzip.File, walkFunc WalkFunc) error {
	zFile, err := f.Open()
	if err != nil {
		err = walkFunc(path.Clean(f.Name), f.FileInfo(), nil, err)
	} else {
		var ra io.ReaderAt
		ra, err = newReaderAt(zFile, f.FileInfo().Size())
		err = walkFunc(path.Clean(f.Name), f.FileInfo(), ra, err)
	}
	defer zFile.Close()
	return err
}
