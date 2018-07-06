// +build windows

package container // import "github.com/docker/docker/daemon/cluster/executor/container"

const (
	testAbsPath        = `c:\foo`
	testAbsNonExistent = `c:\some-non-existing-host-path\`
)

func TestControllerValidateMountNamedPipe(t *testing.T) {
	// with Windows syntax
	if _, err := newTestControllerWithMount(api.Mount{
		Type:   api.MountTypeNamedPipe,
		Source: `\\.\pipe\foo`,
		Target: `\\.\pipe\foo`,
	}); err == nil || !strings.Contains(err.Error(), "invalid volume mount source") {
		t.Fatalf("expected error, got: %v", err)
	}

	// with Unix syntax
	if _, err := newTestControllerWithMount(api.Mount{
		Type:   api.MountTypeNamedPipe,
		Source: `//./pipe/foo`,
		Target: `//./pipe/foo`,
	}); err != nil {
		t.Fatalf("controller should not error at creation: %v", err)
	}
}