package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/garminapi"
)

var gearGroup = Group{Name: "gear", Help: "Manage gear.", Children: []Node{
	Leaf{
		Name: "list", Help: "List all gear.", Default: true,
		Endpoint: "gear.list",
		Params:   Params{"userProfilePK": ProfilePK},
		Verb:     "get gear", Output: JSON{List: true},
	},
	Leaf{
		Name: "stats", Help: "Show usage statistics for a gear item.",
		Spec: argsOf(&struct {
			UUID string `arg:"" help:"Gear UUID."`
		}{}),
		Endpoint: "gear.stats",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "get gear stats", Output: JSON{},
	},
	Leaf{
		Name: "activities", Help: "Show activities linked to a gear item.",
		Spec: argsOf(&struct {
			UUID  string `arg:"" help:"Gear UUID."`
			Limit int    `help:"Maximum number of activities to return." default:"20"`
		}{}),
		Endpoint: "gear.activities",
		Params:   Params{"gearUUID": In("UUID"), "limit": In("Limit")},
		Verb:     "get gear activities", Output: JSON{List: true},
	},
	Leaf{
		Name: "defaults", Help: "Show default gear per activity type.",
		Endpoint: "gear.defaults",
		Params:   Params{"userProfileNumber": ProfilePK},
		Verb:     "get gear defaults", Output: JSON{List: true},
	},
	Leaf{
		Name: "link", Help: "Link gear to an activity.",
		Spec: argsOf(&struct {
			UUID       string `arg:"" help:"Gear UUID."`
			ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
		}{}),
		Endpoint: "gear.link",
		Params:   Params{"gearUUID": In("UUID"), "activityID": In("ActivityID")},
		Verb:     "link gear",
		Output:   Write{Done: "Linked gear {UUID} to activity {ActivityID}"},
	},
	Leaf{
		Name: "unlink", Help: "Unlink gear from an activity.",
		Spec: argsOf(&struct {
			UUID       string `arg:"" help:"Gear UUID."`
			ActivityID string `arg:"" help:"Activity ID." name:"activity-id"`
		}{}),
		Endpoint: "gear.unlink",
		Params:   Params{"gearUUID": In("UUID"), "activityID": In("ActivityID")},
		Verb:     "unlink gear",
		Output:   Write{Done: "Unlinked gear {UUID} from activity {ActivityID}"},
	},
	Leaf{Name: "types", Help: "List gear types.", Endpoint: "gear.types", Verb: "get gear types", Output: JSON{List: true}},
	Custom{Name: "add", Help: "Add a gear item.", Cmd: &GearAddCmd{}},
	Custom{Name: "update", Help: "Change the name, brand, model or notes of a gear item.", Cmd: &GearUpdateCmd{}},
	Leaf{
		Name: "retire", Help: "Retire a gear item.",
		Spec:     gearUUIDArgs,
		Endpoint: "gear.retire",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "retire gear",
		Output:   Write{Done: "Retired gear {UUID}"},
	},
	Leaf{
		Name: "unretire", Help: "Make a retired gear item active again.",
		Spec:     gearUUIDArgs,
		Endpoint: "gear.unretire",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "unretire gear",
		Output:   Write{Done: "Unretired gear {UUID}"},
	},
	Leaf{
		Name: "delete", Help: "Delete a gear item.",
		Spec:     gearUUIDArgs,
		Endpoint: "gear.delete",
		Params:   Params{"gearUUID": In("UUID")},
		Verb:     "delete gear",
		Output:   Write{Done: "Deleted gear {UUID}", Confirm: "Delete gear {UUID}?"},
	},
	gearCollectionsGroup,
}}

var gearCollectionsGroup = Group{Name: "collections", Help: "Manage gear collections.", Children: []Node{
	Leaf{Name: "list", Help: "List gear collections.", Default: true, Endpoint: "gear.collections", Verb: "get gear collections", Output: JSON{List: true}},
	Custom{Name: "create", Help: "Create a gear collection.", Cmd: &GearCollectionCreateCmd{}},
	Custom{Name: "rename", Help: "Rename a gear collection.", Cmd: &GearCollectionRenameCmd{}},
	Leaf{
		Name: "add", Help: "Add a gear item to a gear collection.",
		Spec: argsOf(&struct {
			UUID string   `arg:"" help:"Collection UUID."`
			Gear GearUUID `arg:"" help:"Gear UUID, with or without dashes." name:"gear-uuid"`
		}{}),
		Endpoint: "gear.collection.add-gear",
		Params:   Params{"collectionUUID": In("UUID"), "gearUUID": In("Gear")},
		Verb:     "add gear to collection",
		Output:   Write{Done: "Added gear {Gear} to collection {UUID}"},
	},
	Leaf{
		Name: "delete", Help: "Delete a gear collection. The gear items stay.",
		Spec: argsOf(&struct {
			UUID string `arg:"" help:"Collection UUID."`
		}{}),
		Endpoint: "gear.collection.delete",
		Params:   Params{"collectionUUID": In("UUID")},
		Verb:     "delete gear collection",
		Output:   Write{Done: "Deleted gear collection {UUID}", Confirm: "Delete gear collection {UUID}?"},
	},
}}

var gearUUIDArgs = argsOf(&struct {
	UUID GearUUID `arg:"" help:"Gear UUID, with or without dashes."`
}{})

// GearUUID is a gear UUID in the dashed form that the v2 gear endpoints
// need. gear list prints it without dashes, and the v2 endpoints answer 404
// for that form.
type GearUUID string

var gearUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{12}$`)

func (u *GearUUID) Decode(ctx *kong.DecodeContext) error {
	var raw string
	if err := ctx.Scan.PopValueInto("uuid", &raw); err != nil {
		return err
	}
	if !gearUUIDPattern.MatchString(raw) {
		return fmt.Errorf("invalid gear UUID %q: expected 32 hex digits, with or without dashes", raw)
	}
	hex := strings.ReplaceAll(raw, "-", "")
	*u = GearUUID(strings.Join([]string{hex[:8], hex[8:12], hex[12:16], hex[16:20], hex[20:]}, "-"))
	return nil
}

// gearTypes maps the --type values of gear add to Garmin gear types.
var gearTypes = map[string]string{
	"shoes": "SHOES",
	"bike":  "BIKE",
	"other": "OTHER",
}

// GearAddCmd adds a gear item.
type GearAddCmd struct {
	Name     string `arg:"" help:"Gear name."`
	Type     string `help:"Gear type." default:"shoes" enum:"shoes,bike,other"`
	Brand    string `help:"Brand." required:""`
	Model    string `help:"Model." required:""`
	Notes    string `help:"Notes (1 to 2000 characters). Defaults to the name, because Garmin requires notes."`
	FirstUse Date   `help:"Date of first use (YYYY-MM-DD, today, yesterday, 3d)." default:"today" name:"first-use"`
}

func (c *GearAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.CreateGear(g.Context, garminapi.NewGear{
		Name:         c.Name,
		Type:         gearTypes[c.Type],
		Brand:        c.Brand,
		Model:        c.Model,
		Notes:        c.Notes,
		FirstUseDate: string(c.FirstUse),
	})
	if err != nil {
		return fmt.Errorf("add gear: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if uuid := jsonString(resp, "uuid"); uuid != "" {
		return reportWrite(g, data, "Added gear %q (UUID: %s)", c.Name, uuid)
	}
	return reportWrite(g, data, "Added gear %q", c.Name)
}

// GearUpdateCmd changes fields of a gear item.
type GearUpdateCmd struct {
	UUID  GearUUID `arg:"" help:"Gear UUID, with or without dashes."`
	Name  string   `help:"New name."`
	Brand string   `help:"New brand."`
	Model string   `help:"New model."`
	Notes string   `help:"New notes (1 to 2000 characters)."`
}

func (c *GearUpdateCmd) Validate() error {
	if c.Name == "" && c.Brand == "" && c.Model == "" && c.Notes == "" {
		return errors.New("nothing to change: give --name, --brand, --model or --notes")
	}
	return nil
}

func (c *GearUpdateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateGear(g.Context, string(c.UUID), garminapi.GearEdit{Name: c.Name, Brand: c.Brand, Model: c.Model, Notes: c.Notes})
	if err != nil {
		return fmt.Errorf("update gear: %w", err)
	}
	return reportWrite(g, data, "Updated gear %s", c.UUID)
}

// GearCollectionCreateCmd creates a gear collection.
type GearCollectionCreateCmd struct {
	Name     string `arg:"" help:"Collection name."`
	FirstUse Date   `help:"Date of first use (YYYY-MM-DD, today, yesterday, 3d)." default:"today" name:"first-use"`
}

func (c *GearCollectionCreateCmd) Run(g *Globals) error {
	body, err := json.Marshal(map[string]any{
		"name":                    c.Name,
		"firstUseDate":            string(c.FirstUse),
		"associatedActivityTypes": []any{},
	})
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "gear.collection.create", nil, body)
	if err != nil {
		return fmt.Errorf("create gear collection: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if uuid := jsonString(resp, "uuid"); uuid != "" {
		return reportWrite(g, data, "Created gear collection %q (UUID: %s)", c.Name, uuid)
	}
	return reportWrite(g, data, "Created gear collection %q", c.Name)
}

// GearCollectionRenameCmd renames a gear collection.
type GearCollectionRenameCmd struct {
	UUID string `arg:"" help:"Collection UUID."`
	Name string `arg:"" help:"New name."`
}

func (c *GearCollectionRenameCmd) Validate() error {
	if c.Name == "" {
		return errors.New("the new name must not be empty")
	}
	return nil
}

func (c *GearCollectionRenameCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.RenameGearCollection(g.Context, c.UUID, c.Name)
	if err != nil {
		return fmt.Errorf("rename gear collection: %w", err)
	}
	return reportWrite(g, data, "Renamed gear collection %s to %q", c.UUID, c.Name)
}
