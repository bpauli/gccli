package cmd

import (
	"fmt"
)

var devicesGroup = Group{Name: "devices", Help: "Manage devices.", Children: []Node{
	Leaf{Name: "list", Help: "List registered devices.", Default: true, Endpoint: "devices.list", Verb: "get devices", Output: JSON{List: true}},
	Leaf{
		Name: "settings", Help: "Show settings for a device.",
		Spec: argsOf(&struct {
			DeviceID ID `arg:"" help:"Device ID."`
		}{}),
		Endpoint: "devices.settings",
		Params:   Params{"deviceID": In("DeviceID")},
		Verb:     "get device settings", Output: JSON{},
	},
	Leaf{Name: "primary", Help: "Show primary training device.", Endpoint: "devices.primary-training", Verb: "get primary training device", Output: JSON{}},
	Leaf{
		Name: "solar", Help: "Show solar charging data for a device.",
		Spec: argsOf(&struct {
			DeviceID  ID   `arg:"" help:"Device ID."`
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to start date." name:"end"`
		}{}),
		Endpoint: "devices.solar",
		Params: Params{
			"deviceID":  In("DeviceID"),
			"startDate": In("StartDate").Or(Today),
			"endDate":   In("EndDate").Or(In("StartDate")).Or(Today),
		},
		Verb: "get device solar", Output: JSON{},
	},
	Custom{Name: "alarms", Help: "Show alarms from all devices.", Cmd: &DeviceAlarmsCmd{}},
	Leaf{Name: "last-used", Help: "Show last used device.", Endpoint: "devices.last-used", Verb: "get last used device", Output: JSON{}},
	Leaf{Name: "sensors", Help: "List paired sensors.", Endpoint: "devices.sensors", Verb: "get sensors", Output: JSON{List: true}},
	Leaf{Name: "backups", Help: "Show device backup summary.", Endpoint: "devices.backup-summary", Verb: "get device backup summary", Output: JSON{}},
	Leaf{Name: "historical", Help: "List historical devices.", Endpoint: "devices.historical", Verb: "get historical devices", Output: JSON{List: true}},
	Leaf{Name: "messages", Help: "List device messages.", Endpoint: "devices.messages", Verb: "get device messages", Output: JSON{}},
}}

// DeviceAlarmsCmd shows alarms from all devices.
type DeviceAlarmsCmd struct{}

func (c *DeviceAlarmsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetDeviceAlarms(g.Context)
	if err != nil {
		return fmt.Errorf("get device alarms: %w", err)
	}

	return writeAnswer(g, data, "[]")
}
