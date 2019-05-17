package image // import "github.com/docker/docker/integration/image"

import (
	"archive/tar"
	"bytes"
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
	"gotest.tools/poll"
)

func TestRemoveImageGarbageCollector(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	client := testEnv.APIClient()

	img := "garbage-collector-phase"

	// Build a container image with multiple layers
	content := `FROM busybox
	RUN echo echo Starting application > /start.sh`
	dockerfile := []byte(content)

	buff := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buff)
	assert.NilError(t, tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	}))
	_, err := tw.Write(dockerfile)
	assert.NilError(t, err)
	assert.NilError(t, tw.Close())
	resp, err := client.ImageBuild(ctx, buff, types.ImageBuildOptions{Remove: true,Tags: []string{img},})
	assert.NilError(t, err)
	defer resp.Body.Close()
	image, _, err := client.ImageInspectWithRaw(ctx, img)
	assert.NilError(t, err)

	// Run a container from created image and verify that it starts correctly
	cID := container.Run(t, ctx, client, container.WithImage(img), container.WithCmd(""))
	poll.WaitOn(t, container.IsInState(ctx, client, cID, "running"), poll.WithDelay(100*time.Millisecond))
	res, err := container.Exec(ctx, client, cID,
		[]string{"sh", "/start.sh"})
	assert.NilError(t, err)
	assert.Assert(t, res.Stdout(), "Starting application")

	// Mark latest image layer to immutable
	data := image.GraphDriver.Data
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