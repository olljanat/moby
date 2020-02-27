package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"

	"golang.org/x/sys/windows"
	"gotest.tools/v3/assert"
)

// SignalDaemonDump sends a signal to the daemon to write a dump file
func SignalDaemonDump(pid int) {
	ev, _ := windows.UTF16PtrFromString("Global\\docker-daemon-" + strconv.Itoa(pid))
	h2, err := windows.OpenEvent(0x0002, false, ev)
	if h2 == 0 || err != nil {
		return
	}
	windows.PulseEvent(h2)
}

func signalDaemonReload(pid int) error {
	return fmt.Errorf("daemon reload not supported")
}

func cleanupNetworkNamespace(_ testing.TB, _ *Daemon) {}

// CgroupNamespace returns the cgroup namespace the daemon is running in
func (d *Daemon) CgroupNamespace(t testing.TB) string {
	assert.Assert(t, false)
	return "cgroup namespaces are not supported on Windows"
}

func setsid(cmd *exec.Cmd) {
}

// Sock returns host of the daemon started by hack\ci\windows.ps1
func (d *Daemon) Sock() string {
	return fmt.Sprintf("tcp://127.0.0.1:2357")
}

// On Windows we force leave from swarm instead of stop daemon
func (d *Daemon) Stop(t testing.TB) {
	c := d.NewClientT(t)
	c.SwarmLeave(context.Background(), true)
}
