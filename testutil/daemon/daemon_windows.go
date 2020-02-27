package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

func cleanupMount(_ testing.TB, _ *Daemon) {}

func cleanupNetworkNamespace(_ testing.TB, _ *Daemon) {}

// CgroupNamespace returns the cgroup namespace the daemon is running in
func (d *Daemon) CgroupNamespace(t testing.TB) string {
	assert.Assert(t, false)
	return "cgroup namespaces are not supported on Windows"
}

func setsid(cmd *exec.Cmd) {
}

// Sock returns the npipe path of the daemon
func (d *Daemon) Sock() string {
	return fmt.Sprintf("npipe:////./pipe/docker_engine")
}

// On Windows we do not stop daemon but instead of just cleanup it
// because we want to avoid ingress removal and re-creation process
// which would break network from CI servers.
func (d *Daemon) Stop(t testing.TB) {
	c := d.NewClientT(t)
	c.ContainersPrune(context.Background(), filters.NewArgs())
	c.ImagesPrune(context.Background(), filters.NewArgs())
	c.NetworksPrune(context.Background(), filters.NewArgs())
	c.VolumesPrune(context.Background(), filters.NewArgs())

	serviceList, _ := c.ServiceList(context.Background(), types.ServiceListOptions{})
	for _, service := range serviceList {
		c.ServiceRemove(context.Background(), service.ID)
	}
	configList, _ := c.ConfigList(context.Background(), types.ConfigListOptions{})
	for _, config := range configList {
		c.ConfigRemove(context.Background(), config.ID)
	}
	secretList, _ := c.SecretList(context.Background(), types.SecretListOptions{})
	for _, secret := range secretList {
		c.SecretRemove(context.Background(), secret.ID)
	}
}
