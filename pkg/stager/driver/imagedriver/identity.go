package imagedriver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	zlog "github.com/rs/zerolog/log"
)

var _ csi.IdentityServer = &Driver{}

func (d *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	zlog.Trace().Interface("request", req).Msg("GetPluginInfo called")
	resp := &csi.GetPluginInfoResponse{
		Name:          DriverName,
		VendorVersion: d.vendorVesion,
	}
	return resp, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	zlog.Trace().Interface("request", req).Msg("GetPluginCapabilities called")
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_UNKNOWN,
					},
				},
			},
		},
	}
	return resp, nil
}

func (d *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	zlog.Trace().Interface("request", req).Msg("Probe called")
	return &csi.ProbeResponse{}, nil
}
