package archive

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"path"
	"strings"
)

// WalkTar may be passed a compressed reader instead of an explicit file
func WalkTar(file io.Reader, walkFunc WalkFunc) error {

	tr := tar.NewReader(file)

	for {
		// defines a sub error in the loop scope
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case errors.Is(err, io.EOF):
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		fi := header.FileInfo()

		switch header.Typeflag {
		case tar.TypeLink:
			src := path.Clean(header.Name)
			dst := header.Linkname
			err = walkFunc(src, fi, strings.NewReader(dst), nil)
			if err != nil {
				return err
			}
			continue
		case tar.TypeSymlink:
			err = walkFunc(path.Clean(header.Name), fi, strings.NewReader(header.Linkname), nil)
			if err != nil {
				return err
			}
			continue
		case tar.TypeDir:

			// don't read directories
			err = walkFunc(path.Clean(header.Name), fi, bytes.NewReader(nil), nil)
			if err != nil {
				return err
			}
			continue
		default:
			// read files
			ra, err := newReaderAt(tr, fi.Size())

			// the target location where the dir/file should be created
			err = walkFunc(path.Clean(header.Name), fi, ra, err)
			if err != nil {
				return err
			}

		}
	}
}
