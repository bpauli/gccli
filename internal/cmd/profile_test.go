package cmd

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// The fixtures are the floors that Garmin stored for these FTP values.
func TestDefaultPowerZoneFloors(t *testing.T) {
	tests := []struct {
		sport Upper
		ftp   int
		want  []int
	}{
		{"cycling", 279, []int{0, 153, 209, 251, 293, 335, 419}},
		{"cross-country-skiing", 207, []int{0, 114, 155, 186, 217, 248, 311}},
		{"rowing", 160, []int{0, 88, 120, 144, 168, 192, 240}},
	}
	for _, tt := range tests {
		got := defaultPowerZoneFloors(tt.ftp, powerZoneRules[tt.sport].percents)
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s FTP %d: floors %v, want %v", tt.sport, tt.ftp, got, tt.want)
		}
	}
}

func TestPowerZoneRulesCoverSportEnum(t *testing.T) {
	field, _ := reflect.TypeFor[ProfilePowerZonesSetCmd]().FieldByName("Sport")
	enum := strings.Split(field.Tag.Get("enum"), ",")
	for _, sport := range enum {
		if powerZoneRules[Upper(sport)].zones == 0 {
			t.Errorf("sport %q has no power zone rule", sport)
		}
	}
	if len(powerZoneRules) != len(enum) {
		t.Errorf("powerZoneRules has %d sports, the enum %d", len(powerZoneRules), len(enum))
	}
}
