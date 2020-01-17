package plugin // import "github.com/docker/docker/plugin"

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/daemon/initlayer"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/authorization"
	"github.com/docker/docker/pkg/containerfs"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/mount"
	"github.com/docker/docker/pkg/pubsub"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/system"
	v2 "github.com/docker/docker/plugin/v2"
	"github.com/docker/docker/registry"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const configFileName = "config.json"
const rootFSFileName = "rootfs"

var validFullID = regexp.MustCompile(`^([a-f0-9]{64})$`)

// Executor is the interface that the plugin manager uses to interact with for starting/stopping plugins
type Executor interface {
	Create(id string, spec specs.Spec, stdout, stderr io.WriteCloser) error
	IsRunning(id string) (bool, error)
	Restore(id string, stdout, stderr io.WriteCloser) (alive bool, err error)
	Signal(id string, signal int) error
}

func (pm *Manager) restorePlugin(p *v2.Plugin, c *controller) error {
	if p.IsEnabled() {
		return pm.restore(p, c)
	}
	return nil
}

type eventLogger func(id, name, action string)

// ManagerConfig defines configuration needed to start new manager.
type ManagerConfig struct {
	Store              *Store // remove
	RegistryService    registry.Service
	LiveRestoreEnabled bool // TODO: remove
	LogPluginEvent     eventLogger
	Root               string
	ExecRoot           string
	CreateExecutor     ExecutorCreator
	AuthzMiddleware    *authorization.Middleware
}

// ExecutorCreator is used in the manager config to pass in an `Executor`
type ExecutorCreator func(*Manager) (Executor, error)

// Manager controls the plugin subsystem.
type Manager struct {
	config    ManagerConfig
	mu        sync.RWMutex // protects cMap
	muGC      sync.RWMutex // protects blobstore deletions
	cMap      map[*v2.Plugin]*controller
	blobStore *basicBlobStore
	publisher *pubsub.Publisher
	executor  Executor
}

// controller represents the manager's control on a plugin.
type controller struct {
	restart       bool
	exitChan      chan bool
	timeoutInSecs int
}

// pluginRegistryService ensures that all resolved repositories
// are of the plugin class.
type pluginRegistryService struct {
	registry.Service
}

func (s pluginRegistryService) ResolveRepository(name reference.Named) (repoInfo *registry.RepositoryInfo, err error) {
	repoInfo, err = s.Service.ResolveRepository(name)
	if repoInfo != nil {
		repoInfo.Class = "plugin"
	}
	return
}

// NewManager returns a new plugin manager.
func NewManager(config ManagerConfig) (*Manager, error) {
	if config.RegistryService != nil {
		config.RegistryService = pluginRegistryService{config.RegistryService}
	}
	manager := &Manager{
		config: config,
	}
	for _, dirName := range []string{manager.config.Root, manager.config.ExecRoot, manager.tmpDir()} {
		if err := os.MkdirAll(dirName, 0700); err != nil {
			return nil, errors.Wrapf(err, "failed to mkdir %v", dirName)
		}
	}
	var err error
	manager.executor, err = config.CreateExecutor(manager)
	if err != nil {
		return nil, err
	}

	manager.blobStore, err = newBasicBlobStore(filepath.Join(manager.config.Root, "storage/blobs"))
	if err != nil {
		return nil, err
	}

	manager.cMap = make(map[*v2.Plugin]*controller)
	if err := manager.reload(); err != nil {
		return nil, errors.Wrap(err, "failed to restore plugins")
	}

	manager.publisher = pubsub.NewPublisher(0, 0)
	return manager, nil
}

func (pm *Manager) tmpDir() string {
	return filepath.Join(pm.config.Root, "tmp")
}

// HandleExitEvent is called when the executor receives the exit event
// In the future we may change this, but for now all we care about is the exit event.
func (pm *Manager) HandleExitEvent(id string) error {
	p, err := pm.config.Store.GetV2Plugin(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(pm.config.ExecRoot, id)); err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).WithField("id", id).Error("Could not remove plugin bundle dir")
	}

	pm.mu.RLock()
	c := pm.cMap[p]
	if c.exitChan != nil {
		close(c.exitChan)
		c.exitChan = nil // ignore duplicate events (containerd issue #2299)
	}
	restart := c.restart
	pm.mu.RUnlock()

	if restart {
		pm.enable(p, c, true)
	} else {
		if err := mount.RecursiveUnmount(filepath.Join(pm.config.Root, id)); err != nil {
			return errors.Wrap(err, "error cleaning up plugin mounts")
		}
	}
	return nil
}

func handleLoadError(err error, id string) {
	if err == nil {
		return
	}
	logger := logrus.WithError(err).WithField("id", id)
	if os.IsNotExist(errors.Cause(err)) {
		// Likely some error while removing on an older version of docker
		logger.Warn("missing plugin config, skipping: this may be caused due to a failed remove and requires manual cleanup.")
		return
	}
	logger.Error("error loading plugin, skipping")
}

func (pm *Manager) reload() error { // todo: restore
	dir, err := ioutil.ReadDir(pm.config.Root)
	if err != nil {
		return errors.Wrapf(err, "failed to read %v", pm.config.Root)
	}
	plugins := make(map[string]*v2.Plugin)
	for _, v := range dir {
		if validFullID.MatchString(v.Name()) {
			p, err := pm.loadPlugin(v.Name())
			if err != nil {
				handleLoadError(err, v.Name())
				continue
			}
			plugins[p.GetID()] = p
		} else {
			if validFullID.MatchString(strings.TrimSuffix(v.Name(), "-removing")) {
				// There was likely some error while removing this plugin, let's try to remove again here
				if err := system.EnsureRemoveAll(v.Name()); err != nil {
					logrus.WithError(err).WithField("id", v.Name()).Warn("error while attempting to clean up previously removed plugin")
				}
			}
		}
	}

	pm.config.Store.SetAll(plugins)

	var wg sync.WaitGroup
	wg.Add(len(plugins))
	for _, p := range plugins {
		c := &controller{exitChan: make(chan bool)}
		pm.mu.Lock()
		pm.cMap[p] = c
		pm.mu.Unlock()

		go func(p *v2.Plugin) {
			defer wg.Done()
			if err := pm.restorePlugin(p, c); err != nil {
				logrus.WithError(err).WithField("id", p.GetID()).Error("Failed to restore plugin")
				return
			}

			if p.Rootfs != "" {
				p.Rootfs = filepath.Join(pm.config.Root, p.PluginObj.ID, "rootfs")
			}

			// We should only enable rootfs propagation for certain plugin types that need it.
			for _, typ := range p.PluginObj.Config.Interface.Types {
				if (typ.Capability == "volumedriver" || typ.Capability == "graphdriver") && typ.Prefix == "docker" && strings.HasPrefix(typ.Version, "1.") {
					if p.PluginObj.Config.PropagatedMount != "" {
						propRoot := filepath.Join(filepath.Dir(p.Rootfs), "propagated-mount")

						// check if we need to migrate an older propagated mount from before
						// these mounts were stored outside the plugin rootfs
						if _, err := os.Stat(propRoot); os.IsNotExist(err) {
							rootfsProp := filepath.Join(p.Rootfs, p.PluginObj.Config.PropagatedMount)
							if _, err := os.Stat(rootfsProp); err == nil {
								if err := os.Rename(rootfsProp, propRoot); err != nil {
									logrus.WithError(err).WithField("dir", propRoot).Error("error migrating propagated mount storage")
								}
							}
						}

						if err := os.MkdirAll(propRoot, 0755); err != nil {
							logrus.Errorf("failed to create PropagatedMount directory at %s: %v", propRoot, err)
						}
					}
				}
			}

			pm.save(p)
			requiresManualRestore := !pm.config.LiveRestoreEnabled && p.IsEnabled()

			if requiresManualRestore {
				// if liveRestore is not enabled, the plugin will be stopped now so we should enable it
				if err := pm.enable(p, c, true); err != nil {
					logrus.WithError(err).WithField("id", p.GetID()).Error("failed to enable plugin")
				}
			}
		}(p)
	}
	wg.Wait()
	return nil
}

// Get looks up the requested plugin in the store.
func (pm *Manager) Get(idOrName string) (*v2.Plugin, error) {
	return pm.config.Store.GetV2Plugin(idOrName)
}

func (pm *Manager) loadPlugin(id string) (*v2.Plugin, error) {
	p := filepath.Join(pm.config.Root, id, configFileName)
	dt, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %v", p)
	}
	var plugin v2.Plugin
	if err := json.Unmarshal(dt, &plugin); err != nil {
		return nil, errors.Wrapf(err, "error decoding %v", p)
	}
	return &plugin, nil
}

func (pm *Manager) save(p *v2.Plugin) error {
	pluginJSON, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "failed to marshal plugin json")
	}
	if err := ioutils.AtomicWriteFile(filepath.Join(pm.config.Root, p.GetID(), configFileName), pluginJSON, 0600); err != nil {
		return errors.Wrap(err, "failed to write atomically plugin json")
	}
	return nil
}

// GC cleans up unreferenced blobs. This is recommended to run in a goroutine
func (pm *Manager) GC() {
	pm.muGC.Lock()
	defer pm.muGC.Unlock()

	whitelist := make(map[digest.Digest]struct{})
	for _, p := range pm.config.Store.GetAll() {
		whitelist[p.Config] = struct{}{}
		for _, b := range p.Blobsums {
			whitelist[b] = struct{}{}
		}
	}

	pm.blobStore.gc(whitelist)
}

type logHook struct{ id string }

func (logHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l logHook) Fire(entry *logrus.Entry) error {
	entry.Data = logrus.Fields{"plugin": l.id}
	return nil
}

func makeLoggerStreams(id string) (stdout, stderr io.WriteCloser) {
	logger := logrus.New()
	logger.Hooks.Add(logHook{id})
	return logger.WriterLevel(logrus.InfoLevel), logger.WriterLevel(logrus.ErrorLevel)
}

func validatePrivileges(requiredPrivileges, privileges types.PluginPrivileges) error {
	if !isEqual(requiredPrivileges, privileges, isEqualPrivilege) {
		return errors.New("incorrect privileges")
	}

	return nil
}

func isEqual(arrOne, arrOther types.PluginPrivileges, compare func(x, y types.PluginPrivilege) bool) bool {
	if len(arrOne) != len(arrOther) {
		return false
	}

	sort.Sort(arrOne)
	sort.Sort(arrOther)

	for i := 1; i < arrOne.Len(); i++ {
		if !compare(arrOne[i], arrOther[i]) {
			return false
		}
	}

	return true
}

func isEqualPrivilege(a, b types.PluginPrivilege) bool {
	if a.Name != b.Name {
		return false
	}

	return reflect.DeepEqual(a.Value, b.Value)
}

func configToRootFS(c []byte) (*image.RootFS, error) {
	var pluginConfig types.PluginConfig
	if err := json.Unmarshal(c, &pluginConfig); err != nil {
		return nil, err
	}
	// validation for empty rootfs is in distribution code
	if pluginConfig.Rootfs == nil {
		return nil, nil
	}

	return rootFSFromPlugin(pluginConfig.Rootfs), nil
}

func rootFSFromPlugin(pluginfs *types.PluginConfigRootfs) *image.RootFS {
	rootFS := image.RootFS{
		Type:    pluginfs.Type,
		DiffIDs: make([]layer.DiffID, len(pluginfs.DiffIds)),
	}
	for i := range pluginfs.DiffIds {
		rootFS.DiffIDs[i] = layer.DiffID(pluginfs.DiffIds[i])
	}

	return &rootFS
}

func (pm *Manager) enable(p *v2.Plugin, c *controller, force bool) error {
	p.Rootfs = filepath.Join(pm.config.Root, p.PluginObj.ID, "rootfs")
	if p.IsEnabled() && !force {
		return errors.Wrap(enabledError(p.Name()), "plugin already enabled")
	}
	spec, err := p.InitSpec(pm.config.ExecRoot)
	if err != nil {
		return err
	}

	c.restart = true
	c.exitChan = make(chan bool)

	pm.mu.Lock()
	pm.cMap[p] = c
	pm.mu.Unlock()

	var propRoot string
	if p.PluginObj.Config.PropagatedMount != "" {
		propRoot = filepath.Join(filepath.Dir(p.Rootfs), "propagated-mount")

		if err := os.MkdirAll(propRoot, 0755); err != nil {
			logrus.Errorf("failed to create PropagatedMount directory at %s: %v", propRoot, err)
		}

		// FixMe:
		/*
			if err := mount.MakeRShared(propRoot); err != nil {
				return errors.Wrap(err, "error setting up propagated mount dir")
			}
		*/
	}

	rootFS := containerfs.NewLocalContainerFS(filepath.Join(pm.config.Root, p.PluginObj.ID, rootFSFileName))
	if err := initlayer.Setup(rootFS, idtools.Identity{UID: 0, GID: 0}); err != nil {
		return errors.WithStack(err)
	}

	stdout, stderr := makeLoggerStreams(p.GetID())
	if err := pm.executor.Create(p.GetID(), *spec, stdout, stderr); err != nil {
		if p.PluginObj.Config.PropagatedMount != "" {
			if err := mount.Unmount(propRoot); err != nil {
				logrus.WithField("plugin", p.Name()).WithError(err).Warn("Failed to unmount vplugin propagated mount root")
			}
		}
		return errors.WithStack(err)
	}
	return pm.pluginPostStart(p, c)
}

func (pm *Manager) restore(p *v2.Plugin, c *controller) error {
	stdout, stderr := makeLoggerStreams(p.GetID())
	alive, err := pm.executor.Restore(p.GetID(), stdout, stderr)
	if err != nil {
		return err
	}

	if pm.config.LiveRestoreEnabled {
		if !alive {
			return pm.enable(p, c, true)
		}

		c.exitChan = make(chan bool)
		c.restart = true
		pm.mu.Lock()
		pm.cMap[p] = c
		pm.mu.Unlock()
		return pm.pluginPostStart(p, c)
	}

	if alive {
		// TODO(@cpuguy83): Should we always just re-attach to the running plugin instead of doing this?
		c.restart = false
		shutdownPlugin(p, c.exitChan, pm.executor)
	}

	return nil
}

func (pm *Manager) disable(p *v2.Plugin, c *controller) error {
	if !p.IsEnabled() {
		return errors.Wrap(errDisabled(p.Name()), "plugin is already disabled")
	}

	c.restart = false
	shutdownPlugin(p, c.exitChan, pm.executor)
	pm.config.Store.SetState(p, false)
	return pm.save(p)
}

// Shutdown stops all plugins and called during daemon shutdown.
func (pm *Manager) Shutdown() {
	plugins := pm.config.Store.GetAll()
	for _, p := range plugins {
		pm.mu.RLock()
		c := pm.cMap[p]
		pm.mu.RUnlock()

		if pm.config.LiveRestoreEnabled && p.IsEnabled() {
			logrus.Debug("Plugin active when liveRestore is set, skipping shutdown")
			continue
		}
		if pm.executor != nil && p.IsEnabled() {
			c.restart = false
			shutdownPlugin(p, c.exitChan, pm.executor)
		}
	}
	if err := mount.RecursiveUnmount(pm.config.Root); err != nil {
		logrus.WithError(err).Warn("error cleaning up plugin mounts")
	}
}

func (pm *Manager) upgradePlugin(p *v2.Plugin, configDigest digest.Digest, blobsums []digest.Digest, tmpRootFSDir string, privileges *types.PluginPrivileges) (err error) {
	config, err := pm.setupNewPlugin(configDigest, blobsums, privileges)
	if err != nil {
		return err
	}

	pdir := filepath.Join(pm.config.Root, p.PluginObj.ID)
	orig := filepath.Join(pdir, "rootfs")

	// Make sure nothing is mounted
	// This could happen if the plugin was disabled with `-f` with active mounts.
	// If there is anything in `orig` is still mounted, this should error out.
	if err := mount.RecursiveUnmount(orig); err != nil {
		return errdefs.System(err)
	}

	backup := orig + "-old"
	if err := os.Rename(orig, backup); err != nil {
		return errors.Wrap(errdefs.System(err), "error backing up plugin data before upgrade")
	}

	defer func() {
		if err != nil {
			if rmErr := os.RemoveAll(orig); rmErr != nil && !os.IsNotExist(rmErr) {
				logrus.WithError(rmErr).WithField("dir", backup).Error("error cleaning up after failed upgrade")
				return
			}
			if mvErr := os.Rename(backup, orig); mvErr != nil {
				err = errors.Wrap(mvErr, "error restoring old plugin root on upgrade failure")
			}
			if rmErr := os.RemoveAll(tmpRootFSDir); rmErr != nil && !os.IsNotExist(rmErr) {
				logrus.WithError(rmErr).WithField("plugin", p.Name()).Errorf("error cleaning up plugin upgrade dir: %s", tmpRootFSDir)
			}
		} else {
			if rmErr := os.RemoveAll(backup); rmErr != nil && !os.IsNotExist(rmErr) {
				logrus.WithError(rmErr).WithField("dir", backup).Error("error cleaning up old plugin root after successful upgrade")
			}

			p.Config = configDigest
			p.Blobsums = blobsums
		}
	}()

	if err := os.Rename(tmpRootFSDir, orig); err != nil {
		return errors.Wrap(errdefs.System(err), "error upgrading")
	}

	p.PluginObj.Config = config
	err = pm.save(p)
	return errors.Wrap(err, "error saving upgraded plugin config")
}

// createPlugin creates a new plugin. take lock before calling.
func (pm *Manager) createPlugin(name string, configDigest digest.Digest, blobsums []digest.Digest, rootFSDir string, privileges *types.PluginPrivileges, opts ...CreateOpt) (p *v2.Plugin, err error) {
	if err := pm.config.Store.validateName(name); err != nil { // todo: this check is wrong. remove store
		return nil, errdefs.InvalidParameter(err)
	}

	config, err := pm.setupNewPlugin(configDigest, blobsums, privileges)
	if err != nil {
		return nil, err
	}

	p = &v2.Plugin{
		PluginObj: types.Plugin{
			Name:   name,
			ID:     stringid.GenerateRandomID(),
			Config: config,
		},
		Config:   configDigest,
		Blobsums: blobsums,
	}
	p.InitEmptySettings()
	for _, o := range opts {
		o(p)
	}

	pdir := filepath.Join(pm.config.Root, p.PluginObj.ID)
	if err := os.MkdirAll(pdir, 0700); err != nil {
		return nil, errors.Wrapf(err, "failed to mkdir %v", pdir)
	}

	defer func() {
		if err != nil {
			os.RemoveAll(pdir)
		}
	}()

	if err := os.Rename(rootFSDir, filepath.Join(pdir, rootFSFileName)); err != nil {
		return nil, errors.Wrap(err, "failed to rename rootfs")
	}

	if err := pm.save(p); err != nil {
		return nil, err
	}

	pm.config.Store.Add(p) // todo: remove

	return p, nil
}
