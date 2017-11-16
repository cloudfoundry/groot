package toot

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Toot struct {
	BaseDir string
}

func (t *Toot) Unpack(logger lager.Logger, id, parentID string, layerTar io.Reader) error {
	logger.Info("unpack-info")
	logger.Debug("unpack-debug")

	if _, exists := os.LookupEnv("TOOT_UNPACK_ERROR"); exists {
		return errors.New("unpack-err")
	}

	layerTarContents, err := ioutil.ReadAll(layerTar)
	must(err)
	saveObject(UnpackArgs{ID: id, ParentID: parentID, LayerTarContents: layerTarContents}, t.pathTo(UnpackArgsFileName))
	return nil
}

func (t *Toot) Bundle(logger lager.Logger, id string, layerIDs []string) (specs.Spec, error) {
	logger.Info("bundle-info")
	logger.Debug("bundle-debug")

	if _, exists := os.LookupEnv("TOOT_BUNDLE_ERROR"); exists {
		return specs.Spec{}, errors.New("bundle-err")
	}

	saveObject(BundleArgs{ID: id, LayerIDs: layerIDs}, t.pathTo(BundleArgsFileName))
	return BundleRuntimeSpec, nil
}

const (
	UnpackArgsFileName = "unpack-args"
	BundleArgsFileName = "bundle-args"
)

var (
	BundleRuntimeSpec = specs.Spec{Root: &specs.Root{Path: "toot-rootfs-path"}}
)

type UnpackArgs struct {
	ID               string
	ParentID         string
	LayerTarContents []byte
}

type BundleArgs struct {
	ID       string
	LayerIDs []string
}

func (t *Toot) pathTo(filename string) string {
	return filepath.Join(t.BaseDir, filename)
}

func saveObject(obj interface{}, pathname string) {
	serialisedObj, err := json.Marshal(obj)
	must(err)
	must(ioutil.WriteFile(pathname, serialisedObj, 0600))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
