package foot

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager/v3"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Foot struct {
	BaseDir string
}

func (t *Foot) Unpack(logger lager.Logger, id string, parentIDs []string, layerTar io.Reader) (int64, error) {
	logger.Info("unpack-info")
	logger.Debug("unpack-debug")

	if _, exists := os.LookupEnv("FOOT_UNPACK_ERROR"); exists {
		return 0, errors.New("unpack-err")
	}

	layerTarContents, err := ioutil.ReadAll(layerTar)
	must(err)
	saveObject([]interface{}{
		UnpackArgs{ID: id, ParentIDs: parentIDs, LayerTarContents: layerTarContents},
	}, t.pathTo(UnpackArgsFileName))
	return int64(len(layerTarContents)), nil
}

func (t *Foot) Bundle(logger lager.Logger, id string, layerIDs []string, diskLimit int64) (specs.Spec, error) {
	logger.Info("bundle-info")
	logger.Debug("bundle-debug")

	if _, exists := os.LookupEnv("FOOT_BUNDLE_ERROR"); exists {
		return specs.Spec{}, errors.New("bundle-err")
	}

	saveObject([]interface{}{
		BundleArgs{ID: id, LayerIDs: layerIDs, DiskLimit: diskLimit},
	}, t.pathTo(BundleArgsFileName))
	return BundleRuntimeSpec, nil
}

func (t *Foot) Delete(logger lager.Logger, id string) error {
	logger.Info("delete-info")
	logger.Debug("delete-debug")

	if _, exists := os.LookupEnv("FOOT_BUNDLE_ERROR"); exists {
		return errors.New("delete-err")
	}

	saveObject([]interface{}{
		DeleteArgs{BundleID: id},
	}, t.pathTo(DeleteArgsFileName))
	return nil
}

func (t *Foot) Stats(logger lager.Logger, id string) (groot.VolumeStats, error) {
	logger.Info("stats-info")
	logger.Debug("stats-debug")

	if _, exists := os.LookupEnv("FOOT_STATS_ERROR"); exists {
		return groot.VolumeStats{}, errors.New("stats-err")
	}

	saveObject([]interface{}{
		StatsArgs{ID: id},
	}, t.pathTo(StatsArgsFileName))
	return ReturnedVolumeStats, nil
}

func (t *Foot) WriteMetadata(logger lager.Logger, id string, volumeData groot.ImageMetadata) error {
	logger.Info("write-metadata-info")
	logger.Debug("write-metadata-debug")

	if _, exists := os.LookupEnv("FOOT_WRITE_METADATA_ERROR"); exists {
		return errors.New("write-metadata-err")
	}

	saveObject([]interface{}{
		WriteMetadataArgs{ID: id, VolumeData: volumeData},
	}, t.pathTo(WriteMetadataArgsFileName))
	return nil
}

const (
	UnpackArgsFileName        = "unpack-args"
	BundleArgsFileName        = "bundle-args"
	ExistsArgsFileName        = "exists-args"
	DeleteArgsFileName        = "delete-args"
	StatsArgsFileName         = "stats-args"
	WriteMetadataArgsFileName = "write-metadata-args"
)

var (
	BundleRuntimeSpec   = specs.Spec{Root: &specs.Root{Path: "foot-rootfs-path"}}
	ReturnedVolumeStats = groot.VolumeStats{DiskUsage: groot.DiskUsage{
		TotalBytesUsed:     1234,
		ExclusiveBytesUsed: 12,
	}}
)

type ExistsCalls []ExistsArgs
type ExistsArgs struct {
	LayerID string
}

type DeleteCalls []DeleteArgs
type DeleteArgs struct {
	BundleID string
}

type UnpackCalls []UnpackArgs
type UnpackArgs struct {
	ID               string
	ParentIDs        []string
	LayerTarContents []byte
}

type BundleCalls []BundleArgs
type BundleArgs struct {
	ID        string
	LayerIDs  []string
	DiskLimit int64
}

type StatsCalls []StatsArgs
type StatsArgs struct {
	ID string
}

type WriteMetadataCalls []WriteMetadataArgs
type WriteMetadataArgs struct {
	ID         string
	VolumeData groot.ImageMetadata
}

func (t *Foot) pathTo(filename string) string {
	return filepath.Join(t.BaseDir, filename)
}

func saveObject(obj []interface{}, pathname string) {
	if _, err := os.Stat(pathname); err == nil {
		currentCall := obj[0]
		loadObject(&obj, pathname)
		obj = append(obj, currentCall)
	}

	serialisedObj, err := json.Marshal(obj)
	must(err)
	must(ioutil.WriteFile(pathname, serialisedObj, 0600))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func loadObject(obj *[]interface{}, pathname string) {
	file, err := os.Open(pathname)
	defer file.Close()
	must(err)

	err = json.NewDecoder(file).Decode(obj)
	must(err)
}
