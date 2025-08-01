package container

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
	"github.com/moby/moby/v2/integration/internal/container"
	"github.com/moby/moby/v2/integration/internal/requirement"
	"github.com/moby/moby/v2/testutil"
	"github.com/moby/moby/v2/testutil/daemon"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/skip"
)

// Bring up a daemon with the specified default cgroup namespace mode, and then create a container with the container options
func testRunWithCgroupNs(ctx context.Context, t *testing.T, daemonNsMode string, containerOpts ...func(*container.TestContainerConfig)) (string, string) {
	d := daemon.New(t, daemon.WithDefaultCgroupNamespaceMode(daemonNsMode))
	apiClient := d.NewClientT(t)

	d.StartWithBusybox(ctx, t)
	defer d.Stop(t)

	cID := container.Run(ctx, t, apiClient, containerOpts...)

	daemonCgroup := d.CgroupNamespace(t)
	containerCgroup := container.GetContainerNS(ctx, t, apiClient, cID, "cgroup")
	return containerCgroup, daemonCgroup
}

// Bring up a daemon with the specified default cgroup namespace mode. Create a container with the container options,
// expecting an error with the specified string
func testCreateFailureWithCgroupNs(ctx context.Context, t *testing.T, daemonNsMode string, errStr string, containerOpts ...func(*container.TestContainerConfig)) {
	d := daemon.New(t, daemon.WithDefaultCgroupNamespaceMode(daemonNsMode))
	apiClient := d.NewClientT(t)

	d.StartWithBusybox(ctx, t)
	defer d.Stop(t)
	_, err := container.CreateFromConfig(ctx, apiClient, container.NewTestConfig(containerOpts...))
	assert.ErrorContains(t, err, errStr)
}

func TestCgroupNamespacesRun(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	// When the daemon defaults to private cgroup namespaces, containers launched
	// should be in their own private cgroup namespace by default
	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "private")
	assert.Assert(t, daemonCgroup != containerCgroup)
}

func TestCgroupNamespacesRunPrivileged(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())
	skip.If(t, testEnv.DaemonInfo.CgroupVersion == "2", "on cgroup v2, privileged containers use private cgroupns")

	ctx := testutil.StartSpan(baseContext, t)

	// When the daemon defaults to private cgroup namespaces, privileged containers
	// launched should not be inside their own cgroup namespaces
	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "private", container.WithPrivileged(true))
	assert.Assert(t, daemonCgroup == containerCgroup)
}

func TestCgroupNamespacesRunDaemonHostMode(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	// When the daemon defaults to host cgroup namespaces, containers
	// launched should not be inside their own cgroup namespaces
	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "host")
	assert.Assert(t, daemonCgroup == containerCgroup)
}

func TestCgroupNamespacesRunHostMode(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	// When the daemon defaults to private cgroup namespaces, containers launched
	// with a cgroup ns mode of "host" should not be inside their own cgroup namespaces
	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "private", container.WithCgroupnsMode("host"))
	assert.Assert(t, daemonCgroup == containerCgroup)
}

func TestCgroupNamespacesRunPrivateMode(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	// When the daemon defaults to private cgroup namespaces, containers launched
	// with a cgroup ns mode of "private" should be inside their own cgroup namespaces
	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "private", container.WithCgroupnsMode("private"))
	assert.Assert(t, daemonCgroup != containerCgroup)
}

func TestCgroupNamespacesRunPrivilegedAndPrivate(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	containerCgroup, daemonCgroup := testRunWithCgroupNs(ctx, t, "private", container.WithPrivileged(true), container.WithCgroupnsMode("private"))
	assert.Assert(t, daemonCgroup != containerCgroup)
}

func TestCgroupNamespacesRunInvalidMode(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	// An invalid cgroup namespace mode should return an error on container creation
	errStr := "invalid cgroup namespace mode: invalid"
	testCreateFailureWithCgroupNs(ctx, t, "private", errStr, container.WithCgroupnsMode("invalid"))
}

// Clients before 1.40 expect containers to be created in the host cgroup namespace,
// regardless of the default setting of the daemon, unless running with cgroup v2
func TestCgroupNamespacesRunOlderClient(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType != "linux")
	skip.If(t, testEnv.IsRemoteDaemon())
	skip.If(t, !requirement.CgroupNamespacesEnabled())

	ctx := testutil.StartSpan(baseContext, t)

	d := daemon.New(t, daemon.WithDefaultCgroupNamespaceMode("private"))
	apiClient := d.NewClientT(t, client.WithVersion("1.39"))

	d.StartWithBusybox(ctx, t)
	defer d.Stop(t)

	cID := container.Run(ctx, t, apiClient)

	daemonCgroup := d.CgroupNamespace(t)
	containerCgroup := container.GetContainerNS(ctx, t, apiClient, cID, "cgroup")
	if testEnv.DaemonInfo.CgroupVersion != "2" {
		assert.Assert(t, daemonCgroup == containerCgroup)
	} else {
		assert.Assert(t, daemonCgroup != containerCgroup)
	}
}
