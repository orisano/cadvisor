// +build !linux

package common

import (
	"io/ioutil"
	"os"
	"path"
)

func listDirectories(dirpath string, parent string, recursive bool, output map[string]struct{}) error {
	entries, err := ioutil.ReadDir(dirpath)
	if err != nil {
		// Ignore if this hierarchy does not exist.
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}
	for _, entry := range entries {
		// We only grab directories.
		if entry.IsDir() {
			name := path.Join(parent, entry.Name())
			output[name] = struct{}{}

			// List subcontainers if asked to.
			if recursive {
				err := listDirectories(path.Join(dirpath, entry.Name()), name, true, output)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
