package imagedriver

import (
	"context"
	"fmt"

	"github.com/everpeace/csi-driver-stager/pkg/stager/image"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ csi.NodeServer = &Driver{}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	zlog.Trace().Interface("request", req).Msg("NodePublishVolume called")

	vol, err := d.initVolume(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// publish
	if err := d.stager.StageIn(vol); err != nil {
		zlog.Error().Err(err).Interface("volume", vol).Msg("publishing volume failed. rolling back the publish process.")

		if errRollback := d.stager.RollBackStageIn(vol); errRollback != nil {
			zlog.Error().Err(err).Interface("volume", vol).Msg("rolling back the publish process failed.")
			return nil, status.Error(codes.Internal, err.Error())
		}
		d.deleteVolume(vol.VolumeID)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) initVolume(req *csi.NodePublishVolumeRequest) (*image.Volume, error) {
	vol, err := image.NewVolume(req, d.clock)
	if err != nil {
		return nil, err
	}
	if v, ok := d.statuses[vol.VolumeID]; ok {
		return nil, errors.Errorf("volumeID=%s has not been fully unpublished. phase=%s", v.VolumeID, v.Phase)
	}
	d.statuses[vol.VolumeID] = vol
	return vol, nil
}

func (d *Driver) deleteVolume(volumeID string) {
	delete(d.statuses, volumeID)
}

func (d *Driver) getVolume(volumeID string) *image.Volume {
	return d.statuses[volumeID]
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	zlog.Trace().Interface("request", req).Msg("NodeUnpublishVolume called")

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	vol := d.getVolume(volumeID)
	if vol == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("volumeID=%s is not initialized", volumeID))
	}

	if err := d.stager.StageOut(vol); err != nil {
		zlog.Error().Err(err).Interface("volume", vol).Msg("unpublishing volume failed")
		return nil, status.Error(codes.Internal, err.Error())
	}

	d.deleteVolume(volumeID)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	zlog.Trace().Interface("request", req).Msg("NodeGetInfo called")
	return &csi.NodeGetInfoResponse{
		NodeId: d.nodeID,
	}, nil
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	zlog.Trace().Interface("request", req).Msg("NodeGetCapabilities called")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_UNKNOWN,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
