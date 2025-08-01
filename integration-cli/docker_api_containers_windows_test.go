//go:build windows

package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"

	"github.com/Microsoft/go-winio"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/v2/testutil"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func (s *DockerAPISuite) TestContainersAPICreateMountsBindNamedPipe(c *testing.T) {
	// Create a host pipe to map into the container
	hostPipeName := fmt.Sprintf(`\\.\pipe\docker-cli-test-pipe-%x`, rand.Uint64())
	pc := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;AU)", // Allow all users access to the pipe
	}
	l, err := winio.ListenPipe(hostPipeName, pc)
	if err != nil {
		c.Fatal(err)
	}
	defer l.Close()

	// Asynchronously read data that the container writes to the mapped pipe.
	var b []byte
	ch := make(chan error)
	go func() {
		conn, err := l.Accept()
		if err == nil {
			b, err = io.ReadAll(conn)
			conn.Close()
		}
		ch <- err
	}()

	containerPipeName := `\\.\pipe\docker-cli-test-pipe`
	text := "hello from a pipe"
	cmd := fmt.Sprintf("echo %s > %s", text, containerPipeName)
	name := "test-bind-npipe"

	ctx := testutil.GetContext(c)
	client := testEnv.APIClient()
	_, err = client.ContainerCreate(ctx,
		&container.Config{
			Image: testEnv.PlatformDefaults.BaseImage,
			Cmd:   []string{"cmd", "/c", cmd},
		}, &container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   "npipe",
					Source: hostPipeName,
					Target: containerPipeName,
				},
			},
		},
		nil, nil, name)
	assert.NilError(c, err)

	err = client.ContainerStart(ctx, name, container.StartOptions{})
	assert.NilError(c, err)

	err = <-ch
	assert.NilError(c, err)
	assert.Check(c, is.Equal(text, strings.TrimSpace(string(b))))
}

func mountWrapper(t *testing.T, device, target, mType, options string) error {
	// This should never be called.
	return errors.Errorf("there is no implementation of Mount on this platform")
}
