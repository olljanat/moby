package main

import (
	"github.com/docker/docker/daemon/config"
	"github.com/spf13/pflag"
)

// installConfigFlags adds flags to the pflag.FlagSet to configure the daemon
func installConfigFlags(conf *config.Config, flags *pflag.FlagSet) error {
	// First handle install flags which are consistent cross-platform
	if err := installCommonConfigFlags(conf, flags); err != nil {
		return err
	}

	// Then platform-specific install flags.
	flags.StringVar(&conf.BridgeConfig.FixedCIDR, "fixed-cidr", "", "IPv4 subnet for fixed IPs")
	flags.StringVarP(&conf.BridgeConfig.Iface, "bridge", "b", "", "Attach containers to a virtual switch")
	flags.StringVarP(&conf.SocketGroup, "group", "G", "", "Users or groups that can access the named pipe")

	// TODO: Can be only used with --ip-masq=false
	flags.BoolVar(&conf.BridgeConfig.EnableIPForward, "ip-forward", false, "Enable IP forwarding to bridge and Ethernet adapters")

	return nil
}

// configureCertsDir configures registry.CertsDir() depending on if the daemon
// is running in rootless mode or not. On Windows, it is a no-op.
func configureCertsDir() {}
