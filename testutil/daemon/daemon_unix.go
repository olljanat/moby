//go:build !windows
// +build !windows

package daemon // import "github.com/docker/docker/testutil/daemon"

import (
	"os/exec"
	"syscall"
	"testing"

	"github.com/moby/sys/mount"
	"golang.org/x/sys/unix"
)

// cleanupMount unmounts the daemon root directory, or logs a message if
// unmounting failed.
func cleanupMount(t testing.TB, d *Daemon) {
	t.Helper()
	if err := mount.Unmount(d.Root); err != nil {
		d.log.Logf("[%s] unable to unmount daemon root (%s): %v", d.id, d.Root, err)
	}
}

// SignalDaemonDump sends a signal to the daemon to write a dump file
func SignalDaemonDump(pid int) {
	unix.Kill(pid, unix.SIGQUIT)
}

func signalDaemonReload(pid int) error {
	return unix.Kill(pid, unix.SIGHUP)
}

func setsid(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

// Sock returns the socket path of the daemon
func (d *Daemon) Sock() string {
	return fmt.Sprintf("unix://" + d.sockPath())
}

// Stop will send a SIGINT every second and wait for the daemon to stop.
// If it times out, a SIGKILL is sent.
// Stop will not delete the daemon directory. If a purged daemon is needed,
// instantiate a new one with NewDaemon.
// If an error occurs while starting the daemon, the test will fail.
func (d *Daemon) Stop(t testing.TB) {
	t.Helper()
	err := d.StopWithError()
	if err != nil {
		if err != errDaemonNotStarted {
			t.Fatalf("[%s] error while stopping the daemon: %v", d.id, err)
		} else {
			t.Logf("[%s] daemon is not started", d.id)
		}
	}
}
