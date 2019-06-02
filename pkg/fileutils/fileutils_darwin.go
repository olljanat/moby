package fileutils // import "github.com/docker/docker/pkg/fileutils"

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// GetTotalUsedFds returns the number of used File Descriptors by
// executing `lsof -p PID`
func GetTotalUsedFds() int {
	pid := os.Getpid()

	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return -1
	}

	outputStr := strings.TrimSpace(string(output))

	fds := strings.Split(outputStr, "\n")

	return len(fds) - 1
}

// Chmod implements chmod on platform specific way
func Chmod(name string, fileMode os.FileMode) error {
	return os.Chmod(name, fileMode)
}
