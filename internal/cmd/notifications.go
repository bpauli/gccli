package cmd

import "fmt"

var notificationsGroup = Group{Name: "notifications", Help: "View notifications and change notification preferences.", Children: []Node{
	Leaf{Name: "list", Help: "List notifications.", Default: true, Endpoint: "notifications.list", Verb: "get notifications", Output: JSON{}},
	notificationsPreferencesGroup,
}}

var notificationsPreferencesGroup = Group{Name: "preferences", Help: "Show and change notification preferences.", Children: []Node{
	Leaf{Name: "view", Help: "Show notification preferences.", Default: true, Endpoint: "notifications.preferences", Verb: "get notification preferences", Output: JSON{}},
	Custom{Name: "set", Help: "Turn a notification preference on or off.", Cmd: &NotificationsPreferencesSetCmd{}},
}}

// NotificationsPreferencesSetCmd turns one notification preference on or off.
type NotificationsPreferencesSetCmd struct {
	Key   FreeText `arg:"" help:"Preference key, such as allowEmailWeeklySummary. Run gccli notifications preferences to see the keys."`
	Value string   `arg:"" help:"true turns the preference on, false turns it off." enum:"true,false"`
}

func (c *NotificationsPreferencesSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.SetNotificationPreference(g.Context, string(c.Key), c.Value == "true")
	if err != nil {
		return fmt.Errorf("set notification preference: %w", err)
	}
	return reportWrite(g, data, "Set notification preference %s to %s", c.Key, c.Value)
}
