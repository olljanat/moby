package plugins // import "github.com/docker/docker/pkg/plugins"

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var npipePath = "npipe:////./pipe/"
var	npipePrefix = "docker_plugin_"
var specsPaths = []string{filepath.Join(os.Getenv("programdata"), "docker", "plugins")}

// Scan scans all the plugin paths and returns all the names it found
func scanSocket() ([]string, error) {
	var names []string

	dirEntries, err := ioutil.ReadDir(npipePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "error reading dir entries")
	}

	for _, fi := range dirEntries {
		if !strings.HasPrefix(fi.Name(), npipePrefix) {
			continue
		}
		names = append(names, strings.TrimPrefix(fi.Name(), npipePrefix))
	}
	return names, nil
}

// plugin returns the plugin registered with the given name
func (l *localRegistry) plugin(name string) (*Plugin) {
	pluginPath := filepath.Join(npipePath, name)
	if _, err := os.Stat(pluginPath); err == nil {
		return NewLocalPlugin(name, pluginPath)
	}
	return nil
}
