package plugin // import "github.com/docker/docker/plugin"

import (
	"encoding/json"
	"net"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/plugins"
	v2 "github.com/docker/docker/plugin/v2"
	"github.com/docker/go-connections/sockets"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (pm *Manager) pluginPostStart(p *v2.Plugin, c *controller) error {
	sockAddr := filepath.Join("docker_plugin_", p.GetSocket())
	p.SetTimeout(time.Duration(c.timeoutInSecs) * time.Second)
	addr := &net.UnixAddr{Net: "npipe", Name: sockAddr}
	p.SetAddr(addr)

	if p.Protocol() == plugins.ProtocolSchemeHTTPV1 {
		client, err := plugins.NewClientWithTimeout(addr.Network()+":////./pipe/"+addr.String(), nil, p.Timeout())
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
		// net dial into the npipe to see if someone's listening.
		const defaultTimeout = 10 * time.Second
		conn, err := sockets.DialPipe(sockAddr, defaultTimeout)
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

const shutdownTimeout = 10 * time.Second

func shutdownPlugin(p *v2.Plugin, ec chan bool, executor Executor) {
	pluginID := p.GetID()

	// FixMe: Figure out correct
	err := executor.Signal(pluginID, 0)
	if err != nil {
		logrus.Errorf("Sending SIGTERM to plugin failed with error: %v", err)
	} else {

		timeout := time.NewTimer(shutdownTimeout)
		defer timeout.Stop()

		select {
		case <-ec:
			logrus.Debug("Clean shutdown of plugin")
		case <-timeout.C:
			logrus.Debug("Force shutdown plugin")
			// FixMe: Figure out correct
			if err := executor.Signal(pluginID, 0); err != nil {
				logrus.Errorf("Sending SIGKILL to plugin failed with error: %v", err)
			}

			timeout.Reset(shutdownTimeout)

			select {
			case <-ec:
				logrus.Debug("SIGKILL plugin shutdown")
			case <-timeout.C:
				logrus.WithField("plugin", p.Name).Warn("Force shutdown plugin FAILED")
			}
		}
	}
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

	return config, nil
}
