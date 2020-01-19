package imagedriver

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/everpeace/csi-driver-stager/pkg/stager/image"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ csi.NodeServer = &Driver{}

func (d *Driver) LogWithVolume(parent zerolog.Logger, vol *image.Volume) zerolog.Logger {
	return parent.With().
		Str("VolumeID", vol.VolumeID).
		Interface("PodInfo", vol.PodInfo).
		Interface("StagerSpec", vol.Spec).
		Logger()
}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodePublishVolume").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")

	vol, err := d.initVolume(req)
	if err != nil {
		logger.Error().Interface("context", req.GetVolumeContext()).Err(err).Msg("failed to initialize volume")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger = d.LogWithVolume(logger, vol)
	logger.Debug().Msg("start")

	// publish
	if err := d.stager.StageIn(vol); err != nil {
		logger.Error().Err(err).Msg("failed to stage-in. rolling back...")

		if errRollback := d.stager.RollBackStageIn(vol); errRollback != nil {
			logger.Error().Err(err).Msg("failed to roll back")
			return nil, status.Error(codes.Internal, err.Error())
		}
		logger.Error().Msg("succeeded rolling back")

		d.deleteVolume(vol.VolumeID)

		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.Debug().Msg("succeeded")
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) initVolume(req *csi.NodePublishVolumeRequest) (*image.Volume, error) {
	vol, err := image.NewVolume(req, d.clock, d.defaultStageInImage)
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
	logger := zlog.With().Str("CSIOperation", "NodeUnpublishVolume").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		err := errors.New("Volume ID missing in request")
		logger.Error().Err(err).Msg("invalid argument")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	vol := d.getVolume(volumeID)
	if vol == nil {
		err := errors.Errorf("volumeID=%s is not initialized", volumeID)
		logger.Error().Err(err).Msg("assertion error")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger = d.LogWithVolume(logger, vol)
	logger.Debug().Msg("start")

	if err := d.stager.StageOut(vol); err != nil {
		logger.Error().Err(err).Msg("failed to stage-out")
		return nil, status.Error(codes.Internal, err.Error())
	}

	d.deleteVolume(volumeID)

	logger.Debug().Msg("succeeded")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeGetInfo").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return &csi.NodeGetInfoResponse{
		NodeId:             d.nodeID,
		AccessibleTopology: &csi.Topology{Segments: map[string]string{}},
	}, nil
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeGetCapabilities").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{},
	}, nil
}

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeStageVolume").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeUnstageVolume").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeGetVolumeStats").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	logger := zlog.With().Str("CSIOperation", "NodeExpandVolume").Logger()
	logger.Trace().Interface("request", req).Msg("method called with the request")
	return nil, status.Error(codes.Unimplemented, "")
}
