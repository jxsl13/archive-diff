package archive

import (
	"os"

	"github.com/ulikunitz/xz"
)

func WalkTarXz(file *os.File, walkFunc WalkFunc) error {
	r, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	return WalkTar(r, walkFunc)
}
