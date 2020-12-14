// +build !linux !no_swarmkit

package cluster // import "github.com/docker/docker/daemon/cluster"

import "net"

func (c *Cluster) resolveSystemAddr() (net.IP, error) {
	return c.resolveSystemAddrViaSubnetCheck()
}
