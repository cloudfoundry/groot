package groot

import (
	"fmt"
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	runspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

func (g *Groot) Create(handle string, rootfsURI *url.URL, diskLimit int64, excludeImageFromQuota bool) (runspec.Spec, error) {
	g.Logger = g.Logger.Session("create")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	if diskLimit < 0 {
		return runspec.Spec{}, fmt.Errorf("invalid disk limit: %d", diskLimit)
	}

	imageSpec := imagepuller.ImageSpec{
		ImageSrc:              rootfsURI,
		DiskLimit:             diskLimit,
		ExcludeImageFromQuota: excludeImageFromQuota,
	}

	image, err := g.ImagePuller.Pull(g.Logger, imageSpec)
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "pulling image")
	}

	quota := diskLimit

	if diskLimit != 0 && !excludeImageFromQuota {
		quota = quota - image.BaseImageSize
		if quota <= 0 {
			return runspec.Spec{}, fmt.Errorf("disk limit %d must be larger than image size %d", diskLimit, image.BaseImageSize)
		}
	}

	bundle, err := g.Driver.Bundle(g.Logger.Session("bundle"), handle, image.ChainIDs, quota)
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "creating bundle")
	}

	metadata := VolumeMetadata{BaseImageSize: image.BaseImageSize}
	err = g.Driver.WriteMetadata(g.Logger.Session("write-metadata"), handle, metadata)

	return bundle, err
}
