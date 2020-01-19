package imagedriver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	zlog "github.com/rs/zerolog/log"
)

var _ csi.IdentityServer = &Driver{}

func (d *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	logger := zlog.With().Str("CSIOperation", "GetPluginInfo").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	resp := &csi.GetPluginInfoResponse{
		Name:          DriverName,
		VendorVersion: d.vendorVesion,
	}
	return resp, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	logger := zlog.With().Str("CSIOperation", "GetPluginCapabilities").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
					},
				},
			},
		},
	}
	return resp, nil
}

func (d *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	logger := zlog.With().Str("CSIOperation", "Probe").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return &csi.ProbeResponse{}, nil
}
