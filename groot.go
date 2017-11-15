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

type Groot struct {
	Driver Driver
}

func Run(driver Driver, argv []string) {
	g := Groot{Driver: driver}

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
			Action: func(ctx *cli.Context) error {
				handle := ctx.Args()[1]
				runtimeSpec, err := g.Create(handle)
				if err != nil {
					return err
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
