package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	testdaemon "github.com/docker/docker/internal/test/daemon"
	"github.com/go-check/check"
)

func init() {
	check.Suite(&DockerSuite{})
}

func (s *DockerSuite) OnTimeout(c *check.C) {
	if testEnv.IsRemoteDaemon() {
		return
	}
	path := filepath.Join(os.Getenv("DEST"), "docker.pid")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		c.Fatalf("Failed to get daemon PID from %s\n", path)
	}

	rawPid, err := strconv.ParseInt(string(b), 10, 32)
	if err != nil {
		c.Fatalf("Failed to parse pid from %s: %s\n", path, err)
	}

	daemonPid := int(rawPid)
	if daemonPid > 0 {
		testdaemon.SignalDaemonDump(daemonPid)
	}
}

func (s *DockerSuite) TearDownTest(c *check.C) {
	testEnv.Clean(c)
}
