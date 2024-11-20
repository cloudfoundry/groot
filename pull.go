package groot

import (
	"fmt"

	"code.cloudfoundry.org/groot/imagepuller"
)

func (g *Groot) Pull() error {
	g.Logger = g.Logger.Session("pull")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	_, err := g.ImagePuller.Pull(g.Logger, imagepuller.ImageSpec{})
	return fmt.Errorf("pulling image: %w", err)
}
