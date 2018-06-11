package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	"github.com/urfave/cli"
)

func main() {
	driver := &foot.Foot{}
	driverFlags := []cli.Flag{
		cli.StringFlag{
			Name:        "driver-store",
			Value:       "",
			Usage:       "driver store path",
			Destination: &driver.BaseDir,
		}}
	groot.Run(driver, os.Args, driverFlags, "0.0.1")
}
