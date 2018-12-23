package swarm // import "github.com/docker/docker/integration/swarm"

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/docker/integration/internal/network"
	"github.com/docker/docker/integration/internal/swarm"
	"github.com/docker/docker/internal/test/daemon"
	"github.com/docker/docker/internal/test/request"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/poll"
	"gotest.tools/skip"
)

func TestSwarmRaftQuorum(t *testing.T) {
	var ops = []func(*daemon.Daemon){}

	if daemonEnabled {
		ops = append(ops, daemon.WithInit)
	}
	d := swarm.NewSwarm(t, testEnv, ops...)
	defer d.Stop(t)
	client := d.NewClientT(t)
	defer client.Close()
}

/*
func (s *DockerSwarmSuite) TestAPISwarmRaftQuorum(c *check.C) {
	d1 := s.AddDaemon(c, true, true)
	d2 := s.AddDaemon(c, true, true)
	d3 := s.AddDaemon(c, true, true)

	d1.CreateService(c, simpleTestService)

	d2.Stop(c)

	// make sure there is a leader
	waitAndAssert(c, defaultReconciliationTimeout, d1.CheckLeader, checker.IsNil)

	d1.CreateService(c, simpleTestService, func(s *swarm.Service) {
		s.Spec.Name = "top1"
	})

	d3.Stop(c)

	var service swarm.Service
	simpleTestService(&service)
	service.Spec.Name = "top2"
	cli, err := d1.NewClient()
	c.Assert(err, checker.IsNil)
	defer cli.Close()

	// d1 will eventually step down from leader because there is no longer an active quorum, wait for that to happen
	waitAndAssert(c, defaultReconciliationTimeout, func(c *check.C) (interface{}, check.CommentInterface) {
		_, err = cli.ServiceCreate(context.Background(), service.Spec, types.ServiceCreateOptions{})
		return err.Error(), nil
	}, checker.Contains, "Make sure more than half of the managers are online.")

	d2.StartNode(c)

	// make sure there is a leader
	waitAndAssert(c, defaultReconciliationTimeout, d1.CheckLeader, checker.IsNil)

	d1.CreateService(c, simpleTestService, func(s *swarm.Service) {
		s.Spec.Name = "top3"
	})
}

*/