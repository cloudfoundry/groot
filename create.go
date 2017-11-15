package groot

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (g *Groot) Create(handle string) (specs.Spec, error) {
	return g.Driver.Bundle(handle, []string{})
}
