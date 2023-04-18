package archive

import (
	"compress/gzip"
	"os"
)

func WalkTarGzip(file *os.File, walkFunc WalkFunc) error {

	r, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer r.Close()

	return WalkTar(r, walkFunc)
}
