package cnmallocator

import (
	"github.com/docker/docker/libnetwork/drivers/overlay/ovmanager"
	"github.com/docker/docker/libnetwork/drivers/remote"
	"github.com/docker/docker/libnetwork/drivers/windows/imanager"
	"github.com/docker/docker/libnetwork/drivers/windows/l2brmanager"
	"github.com/docker/docker/libnetwork/drivers/windows/natmanager"
	"github.com/moby/swarmkit/v2/manager/allocator/networkallocator"
)

var initializers = []initializer{
	{remote.Init, "remote"},
	{ovmanager.Init, "overlay"},
	{imanager.Init, "internal"},
	{l2brmanager.Init, "l2bridge"},
	{natmanager.Init, "nat"},
}

// PredefinedNetworks returns the list of predefined network structures
func PredefinedNetworks() []networkallocator.PredefinedNetworkData {
	return nil
}
