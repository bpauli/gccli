package cmd

import (
	"bytes"
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
	Custom{Name: "set-primary-tracker", Help: "Set the primary activity tracker (not the 'devices primary' training device).", Cmd: &DevicesSetPrimaryTrackerCmd{}},
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
	devicesBackupsGroup,
	Leaf{Name: "historical", Help: "List historical devices.", Endpoint: "devices.historical", Verb: "get historical devices", Output: JSON{List: true}},
	devicesMessagesGroup,
}}

var devicesBackupsGroup = Group{Name: "backups", Help: "Show the device backup summary and list device backups.", Children: []Node{
	Leaf{Name: "summary", Help: "Show device backup summary.", Default: true, Endpoint: "devices.backup-summary", Verb: "get device backup summary", Output: JSON{}},
	Leaf{Name: "list", Help: "List device backups.", Endpoint: "devices.backups", Verb: "get device backups", Output: JSON{List: true}},
}}

var devicesMessagesGroup = Group{Name: "messages", Help: "List and cancel device messages.", Children: []Node{
	Leaf{Name: "list", Help: "List device messages.", Default: true, Endpoint: "devices.messages", Verb: "get device messages", Output: JSON{}},
	Custom{Name: "cancel", Help: "Cancel a device message, such as a sent workout, before the device syncs.", Cmd: &DeviceMessageCancelCmd{}},
}}

// DevicesSetPrimaryTrackerCmd makes a device the primary activity tracker.
type DevicesSetPrimaryTrackerCmd struct {
	DeviceID ID `arg:"" help:"Device ID (from 'devices list')."`
}

func (c *DevicesSetPrimaryTrackerCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "devices.set-primary-tracker", map[string]string{"deviceID": string(c.DeviceID)}, nil)
	if err != nil {
		return fmt.Errorf("set primary activity tracker: %w", err)
	}
	return reportWrite(g, data, "Set device %s as the primary activity tracker", c.DeviceID)
}

// DeviceMessageCancelCmd cancels a pending device message.
type DeviceMessageCancelCmd struct {
	ID ID `arg:"" help:"Message ID (from 'devices messages')."`
}

// Run cancels the message. Garmin answers false when there was nothing to
// cancel: the device synced the message, or the ID is unknown.
func (c *DeviceMessageCancelCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "devices.cancel-message", map[string]string{"messageID": string(c.ID)}, nil)
	if err != nil {
		return fmt.Errorf("cancel device message: %w", err)
	}
	if string(bytes.TrimSpace(data)) == "false" {
		return fmt.Errorf("message %s is not pending (already delivered or unknown)", c.ID)
	}
	return reportWrite(g, data, "Cancelled message %s", c.ID)
}

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
