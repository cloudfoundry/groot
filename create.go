package groot

import (
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

func (g *Groot) Create(handle, rootfsURI string) (specs.Spec, error) {
	g.Logger = g.Logger.Session("create")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	layerID, err := g.LayerIDGenerator.GenerateLayerID(rootfsURI)
	if err != nil {
		return specs.Spec{}, errors.Wrap(err, "generating layerID")
	}

	rootfsFile, err := os.Open(rootfsURI)
	if err != nil {
		return specs.Spec{}, errors.Wrapf(err, "opening rootfsURI: %s", rootfsURI)
	}
	defer rootfsFile.Close()

	if !g.Driver.Exists(g.Logger.Session("exists"), layerID) {
		if err = g.Driver.Unpack(g.Logger.Session("unpack"), layerID, "", rootfsFile); err != nil {
			return specs.Spec{}, errors.Wrapf(err, "unpacking layer: %s", layerID)
		}
	}

	bundle, err := g.Driver.Bundle(g.Logger.Session("bundle"), handle, []string{layerID})
	if err != nil {
		return specs.Spec{}, errors.Wrap(err, "creating bundle")
	}

	return bundle, nil
}
