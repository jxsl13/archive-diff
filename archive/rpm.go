package archive

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/cavaliergopher/cpio"
	"github.com/cavaliergopher/rpm"

	"github.com/ulikunitz/xz"
)

func WalkRPM(file io.Reader, walkFunc WalkFunc) error {
	// Read the package headers
	pkg, err := rpm.Read(file)
	if err != nil {
		return err
	}

	// Check the archive format of the payload
	if format := pkg.PayloadFormat(); format != "cpio" {
		return fmt.Errorf("unsupported payload format: %s", format)
	}

	var compReader io.Reader

	switch format := pkg.PayloadCompression(); format {
	case "xz":
		compReader, err = xz.NewReader(file)

	case "gzip":
		compReader, err = gzip.NewReader(file)
	default:
		return fmt.Errorf("unsupported rpm compression format: %s", format)
	}
	if err != nil {
		return err
	}

	// Attach a reader to unarchive each file in the payload
	cpioReader := cpio.NewReader(compReader)
	for {
		// Move to the next file in the archive
		header, err := cpioReader.Next()
		switch {
		// if no more files are found return
		case errors.Is(err, io.EOF):
			return nil

		// return any other error
		case err != nil:
			return err
		}

		fi := header.FileInfo()

		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			err = walkFunc(path.Clean(header.Name), fi, strings.NewReader(header.Linkname), nil)
			if err != nil {
				return err
			}
			continue
		case fi.IsDir():
			err = walkFunc(path.Clean(header.Name), fi, bytes.NewReader(nil), nil)
			if err != nil {
				return err
			}
			continue
		default:
			// read files
			ra, err := newReaderAt(cpioReader, fi.Size())

			// the target location where the dir/file should be created
			err = walkFunc(path.Clean(header.Name), fi, ra, err)
			if err != nil {
				return err
			}
		}
	}

}
