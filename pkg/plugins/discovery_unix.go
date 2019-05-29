// +build !windows

package plugins // import "github.com/docker/docker/pkg/plugins"

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var socketsPath = "/run/docker/plugins"
var specsPaths = []string{"/etc/docker/plugins", "/usr/lib/docker/plugins"}

// Scan scans all the plugin paths and returns all the names it found
func scanSocket() ([]string, error) {
	var names []string
	dirEntries, err := ioutil.ReadDir(socketsPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "error reading dir entries")
	}

	for _, fi := range dirEntries {
		if fi.IsDir() {
			fi, err = os.Stat(filepath.Join(socketsPath, fi.Name(), fi.Name()+".sock"))
			if err != nil {
				continue
			}
		}

		if fi.Mode()&os.ModeSocket != 0 {
			names = append(names, strings.TrimSuffix(filepath.Base(fi.Name()), filepath.Ext(fi.Name())))
		}
	}
	return names, nil
}

// plugin returns the plugin registered with the given name
func (l *localRegistry) plugin(name string) (*Plugin) {
	socketpaths := pluginPaths(socketsPath, name, ".sock")

	for _, p := range socketpaths {
		if fi, err := os.Stat(p); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return NewLocalPlugin(name, "unix://"+p)
		}
	}
	return nil
}
