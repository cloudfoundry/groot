package groot

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (g *Groot) Create(handle string) (specs.Spec, error) {
	runtimeSpec, err := g.Driver.Bundle(handle, []string{})
	if err != nil {
		return specs.Spec{}, fmt.Errorf("driver.Bundle: %s", err)
	}
	return runtimeSpec, nil
}
