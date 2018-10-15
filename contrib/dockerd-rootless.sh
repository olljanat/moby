#!/bin/sh
# dockerd-rootless.sh executes dockerd in rootless mode.
#
# Usage: dockerd-rootless.sh --experimental [DOCKERD_OPTIONS]
# Currently, specifying --experimental is mandatory.
#
# /etc/subuid and /etc/subgid needs to be configured for the current user.
# See the documentation for the further information.

set -e -x
if ! [ -w $XDG_RUNTIME_DIR ]; then
	echo "XDG_RUNTIME_DIR needs to be set and writable"
	exit 1
fi
if ! [ -w $HOME ]; then
	echo "HOME needs to be set and writable"
	exit 1
fi

rootlesskit=""
for f in docker-rootlesskit rootlesskit; do
	if which $f >/dev/null 2>&1; then
		rootlesskit=$f
		break
	fi
done
if [ -z $rootlesskit ]; then
	echo "rootlesskit needs to be installed"
	exit 1
fi

slirp4netns=""
for f in docker-slirp4netns slirp4netns; do
	if which $f >/dev/null 2>&1; then
		slirp4netns=$f
		break
	fi
done
if [ -z $slirp4netns ]; then
	echo "slirp4netns needs to be installed"
	exit 1
fi

if [ -z $_DOCKERD_ROOTLESS_CHILD ]; then
	_DOCKERD_ROOTLESS_CHILD=1
	export _DOCKERD_ROOTLESS_CHILD
	# Re-exec the script via RootlessKit, so as to create unprivileged {user,mount,network} namespaces.
	#
	# --net specifies the network stack. slirp4netns and VPNKit are supported.
	# Currently, slirp4netns is the fastest.
	# See https://github.com/rootless-containers/rootlesskit for the benchmark result.
	#
	# --copy-up allows removing/creating files in the directories by creating tmpfs and symlinks
	# * /etc: copy-up is required so as to prevent `/etc/resolv.conf` in the
	#         namespace from being unexpectedly unmounted when `/etc/resolv.conf` is recreated on the host
	#         (by either systemd-networkd or NetworkManager)
	# * /run: copy-up is required so that we can create /run/docker (hardcoded for plugins) in our namespace
	$rootlesskit \
		--net=slirp4netns --slirp4netns-binary $slirp4netns --mtu=65520 \
		--copy-up=/etc --copy-up=/run \
		$DOCKERD_ROOTLESS_ROOTLESSKIT_FLAGS \
		$0 $@
else
	[ $_DOCKERD_ROOTLESS_CHILD = 1 ]
	# remove the symlinks for the existing files in the parent namespace if any,
	# so that we can create our own files in our mount namespace.
	rm -f /run/docker /run/xtables.lock
	dockerd $@
fi
