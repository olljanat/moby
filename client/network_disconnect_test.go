package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/network"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNetworkDisconnectError(t *testing.T) {
	client := &Client{
		client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}

	err := client.NetworkDisconnect(context.Background(), "network_id", "container_id", false)
	assert.Check(t, is.ErrorType(err, cerrdefs.IsInternal))

	// Empty network ID or container ID
	err = client.NetworkDisconnect(context.Background(), "", "container_id", false)
	assert.Check(t, is.ErrorType(err, cerrdefs.IsInvalidArgument))
	assert.Check(t, is.ErrorContains(err, "value is empty"))

	err = client.NetworkDisconnect(context.Background(), "network_id", "", false)
	assert.Check(t, is.ErrorType(err, cerrdefs.IsInvalidArgument))
	assert.Check(t, is.ErrorContains(err, "value is empty"))
}

func TestNetworkDisconnect(t *testing.T) {
	expectedURL := "/networks/network_id/disconnect"

	client := &Client{
		client: newMockClient(func(req *http.Request) (*http.Response, error) {
			if !strings.HasPrefix(req.URL.Path, expectedURL) {
				return nil, fmt.Errorf("Expected URL '%s', got '%s'", expectedURL, req.URL)
			}

			if req.Method != http.MethodPost {
				return nil, fmt.Errorf("expected POST method, got %s", req.Method)
			}

			var disconnect network.DisconnectOptions
			if err := json.NewDecoder(req.Body).Decode(&disconnect); err != nil {
				return nil, err
			}

			if disconnect.Container != "container_id" {
				return nil, fmt.Errorf("expected 'container_id', got %s", disconnect.Container)
			}

			if !disconnect.Force {
				return nil, fmt.Errorf("expected Force to be true, got %v", disconnect.Force)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			}, nil
		}),
	}

	err := client.NetworkDisconnect(context.Background(), "network_id", "container_id", true)
	assert.NilError(t, err)
}
