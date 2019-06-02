package fileutils // import "github.com/docker/docker/pkg/fileutils"

import (
	"os"

	"github.com/hectane/go-acl"
)

// GetTotalUsedFds Returns the number of used File Descriptors. Not supported
// on Windows.
func GetTotalUsedFds() int {
	return -1
}

// Chmod implements chmod on platform specific way
func Chmod(name string, fileMode os.FileMode) error {
	return acl.Chmod(name, fileMode)
}
