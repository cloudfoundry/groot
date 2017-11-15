package groot

import (
	"encoding/json"
	"fmt"
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

type Driver interface {
	Bundle(id string, layerIDs []string) (specs.Spec, error)
}

func Run(driver Driver, argv []string) {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
			Action: func(ctx *cli.Context) error {
				handle := ctx.Args()[1]

				runtimeSpec, err := driver.Bundle(handle, []string{})
				if err != nil {
					return fmt.Errorf("driver.Bundle: %s", err)
				}

				return json.NewEncoder(os.Stdout).Encode(runtimeSpec)
			},
		},
	}

	if err := app.Run(argv); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
