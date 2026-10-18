package cmd

import (
	"fmt"
)

var devicesGroup = Group{Name: "devices", Help: "Manage devices.", Children: []Node{
	Custom{Name: "list", Help: "List registered devices.", Default: true, Cmd: &DevicesListCmd{}},
	Custom{Name: "settings", Help: "Show settings for a device.", Cmd: &DeviceSettingsCmd{}},
	Custom{Name: "primary", Help: "Show primary training device.", Cmd: &DevicePrimaryCmd{}},
	Custom{Name: "solar", Help: "Show solar charging data for a device.", Cmd: &DeviceSolarCmd{}},
	Custom{Name: "alarms", Help: "Show alarms from all devices.", Cmd: &DeviceAlarmsCmd{}},
	Custom{Name: "last-used", Help: "Show last used device.", Cmd: &DeviceLastUsedCmd{}},
}}

// DevicesListCmd lists registered devices.
type DevicesListCmd struct{}

func (c *DevicesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "devices.list", nil, nil)
	if err != nil {
		return fmt.Errorf("get devices: %w", err)
	}

	return writeHealthJSON(g, data)
}

// DeviceSettingsCmd shows settings for a specific device.
type DeviceSettingsCmd struct {
	DeviceID string `arg:"" help:"Device ID."`
}

func (c *DeviceSettingsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "devices.settings", map[string]string{"deviceID": c.DeviceID}, nil)
	if err != nil {
		return fmt.Errorf("get device settings: %w", err)
	}

	return writeHealthJSON(g, data)
}

// DevicePrimaryCmd shows the primary training device.
type DevicePrimaryCmd struct{}

func (c *DevicePrimaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "devices.primary-training", nil, nil)
	if err != nil {
		return fmt.Errorf("get primary training device: %w", err)
	}

	return writeHealthJSON(g, data)
}

// DeviceSolarCmd shows solar charging data for a device.
type DeviceSolarCmd struct {
	DeviceID  string `arg:"" help:"Device ID."`
	StartDate Date   `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
	EndDate   Date   `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to start date." name:"end"`
}

func (c *DeviceSolarCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	startDate := string(c.StartDate)
	if startDate == "" {
		startDate, err = resolveDate("")
		if err != nil {
			return err
		}
	}

	endDate := string(c.EndDate)
	if endDate == "" {
		endDate = startDate
	}

	data, err := client.Call(g.Context, "devices.solar", map[string]string{"deviceID": c.DeviceID, "startDate": startDate, "endDate": endDate}, nil)
	if err != nil {
		return fmt.Errorf("get device solar: %w", err)
	}

	return writeHealthJSON(g, data)
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

	return writeHealthJSON(g, data)
}

// DeviceLastUsedCmd shows the last used device.
type DeviceLastUsedCmd struct{}

func (c *DeviceLastUsedCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "devices.last-used", nil, nil)
	if err != nil {
		return fmt.Errorf("get last used device: %w", err)
	}

	return writeHealthJSON(g, data)
}
