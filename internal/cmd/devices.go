package cmd

import (
	"fmt"
)

// DevicesCmd groups device subcommands.
type DevicesCmd struct {
	List     DevicesListCmd    `cmd:"" default:"withargs" help:"List registered devices."`
	Settings DeviceSettingsCmd `cmd:"" help:"Show settings for a device."`
	Primary  DevicePrimaryCmd  `cmd:"" help:"Show primary training device."`
	Solar    DeviceSolarCmd    `cmd:"" help:"Show solar charging data for a device."`
	Alarms   DeviceAlarmsCmd   `cmd:"" help:"Show alarms from all devices."`
	LastUsed DeviceLastUsedCmd `cmd:"" name:"last-used" help:"Show last used device."`
}

// DevicesListCmd lists registered devices.
type DevicesListCmd struct{}

func (c *DevicesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetDevices(g.Context)
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

	data, err := client.GetDeviceSettings(g.Context, c.DeviceID)
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

	data, err := client.GetPrimaryTrainingDevice(g.Context)
	if err != nil {
		return fmt.Errorf("get primary training device: %w", err)
	}

	return writeHealthJSON(g, data)
}

// DeviceSolarCmd shows solar charging data for a device.
type DeviceSolarCmd struct {
	DeviceID  string `arg:"" help:"Device ID."`
	StartDate string `help:"Start date (YYYY-MM-DD). Defaults to today." name:"start"`
	EndDate   string `help:"End date (YYYY-MM-DD). Defaults to start date." name:"end"`
}

func (c *DeviceSolarCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	startDate := c.StartDate
	if startDate == "" {
		startDate, err = resolveDate("")
		if err != nil {
			return err
		}
	}

	endDate := c.EndDate
	if endDate == "" {
		endDate = startDate
	}

	data, err := client.GetDeviceSolar(g.Context, c.DeviceID, startDate, endDate)
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

	data, err := client.GetLastUsedDevice(g.Context)
	if err != nil {
		return fmt.Errorf("get last used device: %w", err)
	}

	return writeHealthJSON(g, data)
}
