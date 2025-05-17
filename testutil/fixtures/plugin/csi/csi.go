package main

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"github.com/docker/go-plugins-helpers/volume"
	volumeplugin "github.com/docker/go-plugins-helpers/volume"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
)

type driverServer struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedControllerServer
	csi.UnimplementedNodeServer
}

// Implement CSI interfaces
func (d *driverServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          "csi",
		VendorVersion: "0.1.0",
	}, nil
}
func (d *driverServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}
func (d *driverServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}
func (d *driverServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	return &csi.CreateVolumeResponse{}, nil
}
func (d *driverServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}
func (d *driverServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	return &csi.NodePublishVolumeResponse{}, nil
}
func (d *driverServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// Implement classic volume plugin interfaces
func (d *driverServer) Create(req *volumeplugin.CreateRequest) error {
	ctx := context.Background()
	_, err := d.CreateVolume(ctx, &csi.CreateVolumeRequest{
		Name:       req.Name,
		Parameters: req.Options,
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
		},
	})
	return err
}
func (d *driverServer) List() (*volumeplugin.ListResponse, error) {
	/*
		d.mu.RLock()
		defer d.mu.RUnlock()
	*/

	name := "test"
	var vols []*volumeplugin.Volume
	//for name, _ := range d.volumes {
	vols = append(vols, &volumeplugin.Volume{
		Name: name,
	})
	//}
	return &volumeplugin.ListResponse{Volumes: vols}, nil
}
func (d *driverServer) Get(r *volumeplugin.GetRequest) (*volumeplugin.GetResponse, error) {
	return &volumeplugin.GetResponse{
		Volume: &volumeplugin.Volume{
			Name: r.Name,
		},
	}, nil
}
func (d *driverServer) Remove(r *volumeplugin.RemoveRequest) error {
	ctx := context.Background()
	_, err := d.DeleteVolume(ctx, &csi.DeleteVolumeRequest{
		VolumeId: r.Name,
	})
	return err
}
func (d *driverServer) Path(r *volumeplugin.PathRequest) (*volumeplugin.PathResponse, error) {
	return &volumeplugin.PathResponse{}, nil
}
func (d *driverServer) Mount(r *volumeplugin.MountRequest) (*volumeplugin.MountResponse, error) {
	return &volume.MountResponse{}, nil
}
func (d *driverServer) Unmount(r *volumeplugin.UnmountRequest) error {
	return nil
}
func (d *driverServer) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{Scope: "local"},
	}
}

func main() {
	p, err := filepath.Abs(filepath.Join("run", "docker", "plugins"))
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		panic(err)
	}
	l, err := net.Listen("unix", filepath.Join(p, "csi.sock"))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	ds := &driverServer{}
	csi.RegisterIdentityServer(server, ds)
	csi.RegisterControllerServer(server, ds)
	csi.RegisterNodeServer(server, ds)

	server.Serve(l)
}
