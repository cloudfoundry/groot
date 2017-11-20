package groot

import (
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (g *Groot) Create(handle, rootfsURI string) (specs.Spec, error) {
	layerID, err := g.LayerIDGenerator.GenerateLayerID(rootfsURI)
	if err != nil {
		return specs.Spec{}, err
	}

	rootfsFile, err := os.Open(rootfsURI)
	if err != nil {
		return specs.Spec{}, err
	}
	defer rootfsFile.Close()

	if err := g.Driver.Unpack(g.Logger.Session("unpack"), layerID, "", rootfsFile); err != nil {
		return specs.Spec{}, err
	}

	return g.Driver.Bundle(g.Logger.Session("create"), handle, []string{layerID})
}
