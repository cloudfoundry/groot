package groot

import (
	"encoding/json"
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

type Driver interface {
	Bundle(logger lager.Logger, id string, layerIDs []string) (specs.Spec, error)
}

type Groot struct {
	Driver Driver
	Logger lager.Logger
}

func Run(driver Driver, argv []string) {
	// The `Before` closure sets this. This is ugly, but we don't know the log
	// level until the CLI framework has parsed the flags.
	var g *Groot

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "Path to config file",
		},
	}
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
	app.Before = func(ctx *cli.Context) error {
		conf, err := parseConfig(ctx.GlobalString("config"))
		if err != nil {
			return silentError(err)
		}
		g, err = newGroot(driver, conf)
		if err != nil {
			return silentError(err)
		}
		return nil
	}

	if err := app.Run(argv); err != nil {
		if _, ok := err.(SilentError); !ok {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func newGroot(driver Driver, conf config) (*Groot, error) {
	logger, err := newLogger(conf.LogLevel)
	if err != nil {
		return nil, err
	}
	return &Groot{Driver: driver, Logger: logger}, nil
}

func newLogger(logLevelStr string) (lager.Logger, error) {
	logLevels := map[string]lager.LogLevel{
		"debug": lager.DEBUG,
		"info":  lager.INFO,
		"error": lager.ERROR,
		"fatal": lager.FATAL,
	}

	logLevel, ok := logLevels[logLevelStr]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", logLevelStr)
	}

	logger := lager.NewLogger("groot")
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, logLevel))

	return logger, nil
}

// SilentError silences errors. urfave/cli already prints certain errors, we
// don't want to print them twice
type SilentError struct {
	Underlying error
}

func (e SilentError) Error() string {
	return e.Underlying.Error()
}

func silentError(err error) SilentError {
	return SilentError{Underlying: err}
}
