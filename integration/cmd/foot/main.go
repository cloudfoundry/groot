package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
)

func main() {
	driver := &foot.Foot{BaseDir: os.Getenv("FOOT_BASE_DIR")}
	groot.Run(driver, os.Args)
}
