// Package units holds the unit systems gccli renders table and plain output
// in, and every conversion factor and unit label they use.
package units

import (
	"fmt"
	"math"
	"strings"
)

// System is a concrete unit system. Its zero value is Metric, the fallback
// when no system is pinned and the Garmin profile cannot be read.
type System uint8

const (
	Metric System = iota
	StatuteUS
	StatuteUK
)

var systemNames = [...]string{
	Metric:    "metric",
	StatuteUS: "statute_us",
	StatuteUK: "statute_uk",
}

func (s System) String() string {
	if int(s) >= len(systemNames) {
		return systemNames[Metric]
	}
	return systemNames[s]
}

// ParseSystem accepts a Garmin measurementSystem value, ignoring case and
// surrounding whitespace. "auto" is not a System; see ParsePreference.
func ParseSystem(v string) (System, error) {
	v = strings.TrimSpace(v)
	for i, name := range systemNames {
		if strings.EqualFold(name, v) {
			return System(i), nil
		}
	}
	return Metric, fmt.Errorf("unknown unit system %q (want %s)", v, strings.Join(systemNames[:], ", "))
}

const autoName = "auto"

// Preference is what the user asked for before any lookup: Auto (the zero
// value, read the Garmin profile) or one pinned System.
type Preference struct {
	sys    System
	pinned bool
}

var Auto = Preference{}

func Pin(s System) Preference {
	return Preference{sys: s, pinned: true}
}

// Pinned returns the pinned system and true, or (Metric, false) for Auto.
func (p Preference) Pinned() (System, bool) {
	return p.sys, p.pinned
}

func (p Preference) String() string {
	if !p.pinned {
		return autoName
	}
	return p.sys.String()
}

// PreferenceNames lists every value ParsePreference accepts, "auto" first.
func PreferenceNames() []string {
	return append([]string{autoName}, systemNames[:]...)
}

// ParsePreference accepts "auto", the empty string (as Auto) or any
// ParseSystem value.
func ParsePreference(v string) (Preference, error) {
	if t := strings.TrimSpace(v); t == "" || strings.EqualFold(t, autoName) {
		return Auto, nil
	}
	s, err := ParseSystem(v)
	if err != nil {
		return Auto, err
	}
	return Pin(s), nil
}

const (
	metersPerKilometer = 1000.0
	MetersPerMile      = 1609.344
	metersPerFoot      = 0.3048
)

type scale struct {
	distancePer   float64
	distanceUnit  string
	speedMul      float64
	speedUnit     string
	pacePer       float64
	paceUnit      string
	elevationPer  float64
	elevationUnit string
}

var scales = [...]scale{
	Metric: {
		distancePer: metersPerKilometer, distanceUnit: "km",
		speedMul: 3.6, speedUnit: "km/h",
		pacePer: metersPerKilometer, paceUnit: "/km",
		elevationPer: 1, elevationUnit: "m",
	},
	StatuteUS: {
		distancePer: MetersPerMile, distanceUnit: "mi",
		speedMul: 3600 / MetersPerMile, speedUnit: "mph",
		pacePer: MetersPerMile, paceUnit: "/mi",
		elevationPer: metersPerFoot, elevationUnit: "ft",
	},
	// Garmin Connect shows statute_uk elevation in metres, not feet.
	StatuteUK: {
		distancePer: MetersPerMile, distanceUnit: "mi",
		speedMul: 3600 / MetersPerMile, speedUnit: "mph",
		pacePer: MetersPerMile, paceUnit: "/mi",
		elevationPer: 1, elevationUnit: "m",
	},
}

func (s System) scale() scale {
	if int(s) >= len(scales) {
		return scales[Metric]
	}
	return scales[s]
}

// truncate absorbs float error: 1000 / (1000 / 480) is 479.99999999999994
// in float64 and must show 8:00.
func truncate(v float64) int {
	return int(v + 1e-9)
}

// Distance formats meters, for example "5.12 km" or "3.18 mi". 0 renders "-".
func (s System) Distance(meters float64) string {
	if meters == 0 {
		return "-"
	}
	sc := s.scale()
	return fmt.Sprintf("%.2f %s", meters/sc.distancePer, sc.distanceUnit)
}

// Speed formats meters per second, for example "10.0 km/h" or "6.2 mph". 0 renders "-".
func (s System) Speed(mps float64) string {
	if mps == 0 {
		return "-"
	}
	sc := s.scale()
	return fmt.Sprintf("%.1f %s", mps*sc.speedMul, sc.speedUnit)
}

// Pace formats meters per second as time per distance unit, for example
// "5:51 /km" or "9:25 /mi". 0 renders "-".
func (s System) Pace(mps float64) string {
	if mps == 0 {
		return "-"
	}
	sc := s.scale()
	total := truncate(sc.pacePer / mps)
	return fmt.Sprintf("%d:%02d %s", total/60, total%60, sc.paceUnit)
}

// Elevation formats meters, for example "85 m" or "279 ft". 0 renders "-".
// It rounds to the nearest integer, as Garmin Connect does: 266 m is "873 ft".
func (s System) Elevation(meters float64) string {
	if meters == 0 {
		return "-"
	}
	sc := s.scale()
	return fmt.Sprintf("%d %s", int(math.Round(meters/sc.elevationPer)), sc.elevationUnit)
}
