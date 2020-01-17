package util

import (
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	"k8s.io/utils/mount"
)

func MountTargetPath(source, target string, options []string) error {
	mounter := mount.New("")
	if err := mounter.Mount(source, target, "", options); err != nil {
		zlog.Error().
			Str("source", source).
			Str("target", target).
			Msg("mount failed")
		return errors.Wrapf(err, "mount %s to %s with options=%s failed", source, target, options)
	}
	zlog.Debug().
		Str("source", source).
		Str("target", target).
		Msg("mounted")
	return nil
}

func UnmountTargetPath(target string) error {
	notMnt, err := mount.New("").IsLikelyNotMountPoint(target)
	if err != nil {
		zlog.Debug().
			Str("target", target).
			Msg("mount failed")
		return errors.Wrapf(err, "unmount %s failed", target)
	}
	if !notMnt {
		err := mount.New("").Unmount(target)
		if err != nil {
			zlog.Debug().
				Str("target", target).
				Msg("mount failed")
			return errors.Wrapf(err, "unmount %s failed", target)
		}
	}
	zlog.Debug().
		Str("target", target).
		Msg("unmounted")
	return nil
}
