package main

import (
	"context"
	"path"
	"time"

	"github.com/docker/docker/integration-cli/checker"
	"github.com/docker/docker/internal/test/fixtures/plugin"
	"github.com/docker/docker/internal/test/registry"
	"github.com/go-check/check"
)

func init() {
	check.Suite(&DockerPluginSuite{
		ds: &DockerSuite{},
	})
}

type DockerPluginSuite struct {
	ds       *DockerSuite
	registry *registry.V2
}

func (ps *DockerPluginSuite) registryHost() string {
	return privateRegistryURL
}

func (ps *DockerPluginSuite) getPluginRepo() string {
	return path.Join(ps.registryHost(), "plugin", "basic")
}
func (ps *DockerPluginSuite) getPluginRepoWithTag() string {
	return ps.getPluginRepo() + ":" + "latest"
}

func (ps *DockerPluginSuite) SetUpSuite(c *check.C) {
	testRequires(c, DaemonIsLinux, RegistryHosting)
	ps.registry = registry.NewV2(c)
	ps.registry.WaitReady(c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := plugin.CreateInRegistry(ctx, ps.getPluginRepo(), nil)
	c.Assert(err, checker.IsNil, check.Commentf("failed to create plugin"))
}

func (ps *DockerPluginSuite) TearDownSuite(c *check.C) {
	if ps.registry != nil {
		ps.registry.Close()
	}
}

func (ps *DockerPluginSuite) TearDownTest(c *check.C) {
	ps.ds.TearDownTest(c)
}

func (ps *DockerPluginSuite) OnTimeout(c *check.C) {
	ps.ds.OnTimeout(c)
}
