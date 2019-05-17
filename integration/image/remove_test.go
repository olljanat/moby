package image // import "github.com/docker/docker/integration/image"

import (
	"context"
	"os"
	"io/ioutil"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/daemon/images"
	"github.com/docker/docker/integration/internal/container"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/poll"
	"gotest.tools/skip"
)

func TestRemoveImageOrphaning(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType == "windows", "FIXME")
	defer setupTest(t)()
	ctx := context.Background()
	client := testEnv.APIClient()

	img := "test-container-orphaning"

	// Create a container from busybox, and commit a small change so we have a new image
	cID1 := container.Create(t, ctx, client, container.WithCmd(""))
	commitResp1, err := client.ContainerCommit(ctx, cID1, types.ContainerCommitOptions{
		Changes:   []string{`ENTRYPOINT ["true"]`},
		Reference: img,
	})
	assert.NilError(t, err)

	// verifies that reference now points to first image
	resp, _, err := client.ImageInspectWithRaw(ctx, img)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(resp.ID, commitResp1.ID))

	// Create a container from created image, and commit a small change with same reference name
	cID2 := container.Create(t, ctx, client, container.WithImage(img), container.WithCmd(""))
	commitResp2, err := client.ContainerCommit(ctx, cID2, types.ContainerCommitOptions{
		Changes:   []string{`LABEL Maintainer="Integration Tests"`},
		Reference: img,
	})
	assert.NilError(t, err)

	// verifies that reference now points to second image
	resp, _, err = client.ImageInspectWithRaw(ctx, img)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(resp.ID, commitResp2.ID))

	// try to remove the image, should not error out.
	_, err = client.ImageRemove(ctx, img, types.ImageRemoveOptions{})
	assert.NilError(t, err)

	// check if the first image is still there
	resp, _, err = client.ImageInspectWithRaw(ctx, commitResp1.ID)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(resp.ID, commitResp1.ID))

	// check if the second image has been deleted
	_, _, err = client.ImageInspectWithRaw(ctx, commitResp2.ID)
	assert.Check(t, is.ErrorContains(err, "No such image:"))
}

func TestRemoveImageGarbageCollector(t *testing.T) {
	skip.If(t, testEnv.DaemonInfo.OSType == "windows", "FIXME")
	defer setupTest(t)()
	ctx := context.Background()
	client := testEnv.APIClient()

	img := "garbage-collector-phase"

	// Create a container from busybox, and commit a small change so we have a new image
	cID1 := container.Create(t, ctx, client, container.WithCmd(""))
	commitResp, err := client.ContainerCommit(ctx, cID1, types.ContainerCommitOptions{
		Changes:   []string{`RUN echo echo Migrating database > /migrate.sh`},
		Reference: img,
	})
	assert.NilError(t, err)

	// Create a container from created image, and commit a small change with same reference name
	cID2 := container.Create(t, ctx, client, container.WithImage(img), container.WithCmd(""))
	_, err = client.ContainerCommit(ctx, cID2, types.ContainerCommitOptions{
		Changes:   []string{`RUN echo sh /migrate.sh > /start.sh`},
		Reference: img,
	})
	assert.NilError(t, err)

	// Run a container from created image and verify that database migration script
	// will run before application starts 
	cID := container.Run(t, ctx, client, container.WithImage(img), container.WithCmd(""))
	poll.WaitOn(t, container.IsInState(ctx, client, cID, "running"), poll.WithDelay(100*time.Millisecond))
	res, err := container.Exec(ctx, client, cID,
		[]string{"sh", "/start.sh"})
	assert.Assert(t, res.Stdout(), "Migrating database")

	// Mark layer created on first phase to immutable
	resp, _, err := client.ImageInspectWithRaw(ctx, commitResp.ID)
	data := resp.GraphDriver.Data
	file, _ := os.Open(data["UpperDir"])
	attr := 0x00000010
	fsflags := uintptr(0x40086602)
	argp := uintptr(unsafe.Pointer(&attr))
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), fsflags, argp)

	// Try to remove the image, it should generate error
	// but marking layer back to mutable before checking errors (so we don't break CI server)
	_, err = client.ImageRemove(ctx, img, types.ImageRemoveOptions{})
	attr = 0x00000000
	argp = uintptr(unsafe.Pointer(&attr))
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), fsflags, argp)
	assert.ErrorContains(t, err, "operation not permitted")

	// Verify that layer remaining on disk
	_, err = ioutil.ReadDir(data["UpperDir"])
	if !os.IsNotExist(err) {
		err = nil
	}
	assert.NilError(t, err)

	// Run imageService.Cleanup() and make sure that layer was removed from disk
	i := images.NewImageService(images.ImageServiceConfig{})
	i.Cleanup()
	_, err = ioutil.ReadDir(data["UpperDir"])
	if os.IsNotExist(err) {
		err = nil
	}
	assert.NilError(t, err)
}