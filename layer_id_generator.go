package groot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

type LocalLayerIDGenerator struct {
	ModTimer ModTimer
}

//go:generate counterfeiter . ModTimer
type ModTimer interface {
	ModTime(pathname string) (time.Time, error)
}

func (l *LocalLayerIDGenerator) GenerateLayerID(pathname string) (string, error) {
	modTime, err := l.ModTimer.ModTime(pathname)
	if err != nil {
		return "", err
	}

	layerID := sha256.Sum256([]byte(
		fmt.Sprintf("%s-%d", pathname, modTime.UnixNano()),
	))
	layerIDHex := hex.EncodeToString(layerID[:])

	return layerIDHex, nil
}

type statModTimer struct{}

func (statModTimer) ModTime(pathname string) (time.Time, error) {
	fileInfo, err := os.Stat(pathname)
	if err != nil {
		return time.Time{}, err
	}

	return fileInfo.ModTime(), nil
}
