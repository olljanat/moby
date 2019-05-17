// +build windows

package image // import "github.com/docker/docker/integration/image"

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/integration/internal/container"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/skip"
)

func TestRemoveImageGarbageCollector(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	client := testEnv.APIClient()

	img := "test-container-garbage-collector"

	// Create a container from busybox, and commit a small change so we have a new image
	cID1 := container.Create(t, ctx, client, container.WithCmd(""))
	commitResp1, err := client.ContainerCommit(ctx, cID1, types.ContainerCommitOptions{
		Changes:   []string{`RUN echo @echo off > /migrate.cmd && echo echo Migrating database >> /migrate.cmd`},
		Reference: img,
	})
	assert.NilError(t, err)

	// Create a container from created image, and commit a small change with same reference name
	cID2 := container.Create(t, ctx, client, container.WithImage(img), container.WithCmd(""))
	commitResp2, err := client.ContainerCommit(ctx, cID2, types.ContainerCommitOptions{
		Changes:   []string{`RUN echo @echo off > /start.cmd && echo start /b /wait migrate.cmd >> /start.cmd`},
		Reference: img,
	})
	assert.NilError(t, err)



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
