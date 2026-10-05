package cmd

import (
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/units"
)

func resolveUnits(g *Globals, client *garminapi.Client) units.System {
	if sys, ok := g.Units.Pinned(); ok {
		return sys
	}
	sys, err := client.GetMeasurementSystem(g.Context)
	if err != nil {
		g.UI.Warnf("could not read Garmin unit preference (%v); using metric. Set --units to skip this lookup.", err)
		return units.Metric
	}
	return sys
}
