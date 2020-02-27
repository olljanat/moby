// +build windows

package daemon

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

// On Windows we skip starting daemon and use the one started by hack\ci\windows.ps1 intead of.
func (d *Daemon) StartAndSwarmInit(t testing.TB) {
	d.SwarmInit(t, swarm.InitRequest{})
}
