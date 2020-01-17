package plugin // import "github.com/docker/docker/plugin"

import (
	"encoding/json"
	"net"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/plugins"
	v2 "github.com/docker/docker/plugin/v2"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (pm *Manager) pluginPostStart(p *v2.Plugin, c *controller) error {
	sockAddr := filepath.Join(pm.config.ExecRoot, p.GetID(), p.GetSocket())
	p.SetTimeout(time.Duration(c.timeoutInSecs) * time.Second)
	addr := &net.UnixAddr{Net: "unix", Name: sockAddr}
	p.SetAddr(addr)

	if p.Protocol() == plugins.ProtocolSchemeHTTPV1 {
		client, err := plugins.NewClientWithTimeout(addr.Network()+"://"+addr.String(), nil, p.Timeout())
		if err != nil {
			c.restart = false
			shutdownPlugin(p, c.exitChan, pm.executor)
			return errors.WithStack(err)
		}

		p.SetPClient(client)
	}

	// Initial sleep before net Dial to allow plugin to listen on socket.
	time.Sleep(500 * time.Millisecond)
	maxRetries := 3
	var retries int
	for {
		// net dial into the unix socket to see if someone's listening.
		conn, err := net.Dial("unix", sockAddr)
		if err == nil {
			conn.Close()
			break
		}

		time.Sleep(3 * time.Second)
		retries++

		if retries > maxRetries {
			logrus.Debugf("error net dialing plugin: %v", err)
			c.restart = false
			// While restoring plugins, we need to explicitly set the state to disabled
			pm.config.Store.SetState(p, false)
			shutdownPlugin(p, c.exitChan, pm.executor)
			return err
		}

	}
	pm.config.Store.SetState(p, true)
	pm.config.Store.CallHandler(p)

	return pm.save(p)
}

func (pm *Manager) setupNewPlugin(configDigest digest.Digest, blobsums []digest.Digest, privileges *types.PluginPrivileges) (types.PluginConfig, error) {
	configRC, err := pm.blobStore.Get(configDigest)
	if err != nil {
		return types.PluginConfig{}, err
	}
	defer configRC.Close()

	var config types.PluginConfig
	dec := json.NewDecoder(configRC)
	if err := dec.Decode(&config); err != nil {
		return types.PluginConfig{}, errors.Wrapf(err, "failed to parse config")
	}
	if dec.More() {
		return types.PluginConfig{}, errors.New("invalid config json")
	}

	requiredPrivileges := computePrivileges(config)
	if privileges != nil {
		if err := validatePrivileges(requiredPrivileges, *privileges); err != nil {
			return types.PluginConfig{}, err
		}
	}

	return config, nil
}
