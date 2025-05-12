package convert

import (
	"net"
	"time"

	"github.com/docker/docker/pkg/plugingetter"
	"github.com/moby/swarmkit/v2/node/plugin"
)

// SwarmPluginGetter adapts a plugingetter.PluginGetter to a Swarmkit plugin.Getter.
func SwarmPluginGetter(pg plugingetter.PluginGetter) plugin.Getter {
	return pluginGetter{pg}
}

type pluginGetter struct {
	pg plugingetter.PluginGetter
}

var _ plugin.Getter = (*pluginGetter)(nil)

type swarmPlugin struct {
	plugingetter.CompatPlugin
}

// PluginAddr is a plugin that exposes the socket address for creating custom clients rather than the built-in `*plugins.Client`
type PluginAddr interface {
	Addr() net.Addr
	Timeout() time.Duration
	Protocol() string
}

func (p swarmPlugin) Client() plugin.Client {
	return p.CompatPlugin.Client()
}

func (g pluginGetter) Get(name string, capability string) (plugin.Plugin, error) {
	p, err := g.pg.Get(name, capability, plugingetter.Lookup)
	if err != nil {
		return nil, err
	}
	return swarmPlugin{p}, nil
}

func (g pluginGetter) GetAllManagedPluginsByCap(capability string) []plugin.Plugin {
	pp := g.pg.GetAllManagedPluginsByCap(capability)
	ret := make([]plugin.Plugin, len(pp))
	for i, p := range pp {
		ret[i] = swarmPlugin{p}
	}
	return ret
}
