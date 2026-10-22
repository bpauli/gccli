package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bpauli/gccli/internal/garminapi"
)

var notificationsGroup = Group{Name: "notifications", Help: "View and dismiss notifications, and change notification preferences.", Children: []Node{
	Leaf{
		Name: "list", Help: "List notifications.", Default: true,
		Spec: argsOf(&struct {
			Limit  int    `help:"Maximum number of notifications to return." default:"50"`
			Cursor string `help:"Start after this cursor. Use pagination.nextCursor from the JSON output of the last page."`
		}{}),
		Endpoint: "notifications.list",
		Params:   Params{"limit": In("Limit"), "cursor": In("Cursor")},
		Verb:     "get notifications", Output: JSON{},
	},
	Leaf{Name: "unread", Help: "Show whether there are unread notifications.", Endpoint: "notifications.unread", Verb: "get unread notifications", Output: JSON{}},
	Custom{Name: "dismiss", Help: "Dismiss a notification. Use 'notifications undismiss' to bring it back.", Cmd: &NotificationsDismissCmd{}},
	Custom{Name: "undismiss", Help: "Bring back a dismissed notification.", Cmd: &NotificationsUndismissCmd{}},
	notificationsPreferencesGroup,
}}

var notificationsPreferencesGroup = Group{Name: "preferences", Help: "Show and change notification preferences.", Children: []Node{
	Leaf{Name: "view", Help: "Show notification preferences.", Default: true, Endpoint: "notifications.preferences", Verb: "get notification preferences", Output: JSON{}},
	Custom{Name: "set", Help: "Turn a notification preference on or off.", Cmd: &NotificationsPreferencesSetCmd{}},
}}

// NotificationsDismissCmd dismisses a notification.
type NotificationsDismissCmd struct {
	ID ID `arg:"" help:"Notification ID (from 'notifications list')."`
}

func (c *NotificationsDismissCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := setNotificationDismissed(g.Context, client, c.ID, true)
	if err != nil {
		return fmt.Errorf("dismiss notification: %w", err)
	}
	return reportWrite(g, data, "Dismissed notification %s", c.ID)
}

// NotificationsUndismissCmd brings back a dismissed notification.
type NotificationsUndismissCmd struct {
	ID ID `arg:"" help:"Notification ID (from 'notifications list')."`
}

func (c *NotificationsUndismissCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := setNotificationDismissed(g.Context, client, c.ID, false)
	if err != nil {
		return fmt.Errorf("undismiss notification: %w", err)
	}
	return reportWrite(g, data, "Restored notification %s", c.ID)
}

func setNotificationDismissed(ctx context.Context, client *garminapi.Client, id ID, dismissed bool) (json.RawMessage, error) {
	return client.Call(ctx, "notifications.dismiss", map[string]string{"notificationID": string(id), "dismiss": strconv.FormatBool(dismissed)}, nil)
}

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
