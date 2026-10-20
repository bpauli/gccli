package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/garminapi"
)

var wellnessMenstrualLogGroup = Group{Name: "menstrual-log", Help: "Show, set and delete the menstrual daily log.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show the menstrual daily log of a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "wellness.menstrual-log",
		Params:   Params{"date": In("Date")},
		Verb:     "get menstrual log", Output: JSON{},
	},
	Custom{Name: "set", Help: "Change the menstrual daily log of a date. Other fields keep their values.", Cmd: &WellnessMenstrualLogSetCmd{}},
	Custom{Name: "delete", Help: "Delete the menstrual daily log of a date.", Cmd: &WellnessMenstrualLogDeleteCmd{}},
}}

// menstrualValues lists Garmin's values of each enum field of the menstrual
// daily log, as the Garmin Connect web app sends them. The flags take each
// value in lower-case kebab form, and the Kong var menstrual_<field> lists
// those forms.
var menstrualValues = map[string][]string{
	"flow":            {"LIGHT", "MEDIUM", "HEAVY"},
	"symptoms":        {"ACNE", "BACKACHE", "BLOATING", "BODY_ACHES", "CONSTIPATION", "CRAMPS", "CRAVINGS", "DIARRHEA", "DIZZINESS", "FATIGUE", "HEADACHE", "INSOMNIA", "NAUSEA", "TENDER_BREASTS"},
	"moods":           {"ENERGETIC", "EMOTIONAL", "FINE", "FRUSTRATED", "HAPPY", "IRRITABLE", "MOOD_SWINGS", "MOTIVATED", "OVERWHELMED", "SAD"},
	"discharge":       {"CREAMY", "EGG_WHITE", "NO_DISCHARGE", "SPOTTING", "STICKY", "UNUSUAL"},
	"sex_drive":       {"AVERAGE", "HIGH", "LOW"},
	"sexual_activity": {"PROTECTED", "UNPROTECTED"},
}

func menstrualFlagValue(garmin string) string {
	return strings.ToLower(strings.ReplaceAll(garmin, "_", "-"))
}

func menstrualVars() kong.Vars {
	vars := kong.Vars{}
	for field, values := range menstrualValues {
		var flags []string
		for _, v := range values {
			flags = append(flags, menstrualFlagValue(v))
		}
		vars["menstrual_"+field] = strings.Join(flags, ",")
	}
	return vars
}

// garminMenstrualValue returns Garmin's value for the flag value v of field.
// Kong has checked v against the enum, and the empty value stays empty.
func garminMenstrualValue(field, v string) string {
	for _, g := range menstrualValues[field] {
		if menstrualFlagValue(g) == v {
			return g
		}
	}
	return ""
}

// WellnessMenstrualLogSetCmd sets fields of the menstrual daily log of a day.
// The enum flags with a trailing comma in enum take the empty value, which
// clears the field. Kong leaves a list flag nil when it is not given and
// makes it empty for "", so nil keeps the list and empty clears it.
type WellnessMenstrualLogSetCmd struct {
	Date           Date     `arg:"" help:"Date of the log (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	Flow           *string  `help:"Flow: light, medium or heavy. An empty value clears it." enum:"${menstrual_flow}," placeholder:"FLOW"`
	Symptoms       []string `help:"Comma-separated symptoms: acne, backache, bloating, body-aches, constipation, cramps, cravings, diarrhea, dizziness, fatigue, headache, insomnia, nausea, tender-breasts. An empty value clears them." enum:"${menstrual_symptoms}" placeholder:"SYMPTOM"`
	Moods          []string `help:"Comma-separated moods: energetic, emotional, fine, frustrated, happy, irritable, mood-swings, motivated, overwhelmed, sad. An empty value clears them." enum:"${menstrual_moods}" placeholder:"MOOD"`
	Discharge      []string `help:"Comma-separated discharge: creamy, egg-white, no-discharge, spotting, sticky, unusual. no-discharge takes no other value. An empty value clears it." enum:"${menstrual_discharge}" placeholder:"DISCHARGE"`
	SexDrive       *string  `help:"Sex drive: average, high or low. An empty value clears it." enum:"${menstrual_sex_drive}," placeholder:"LEVEL"`
	SexualActivity *string  `help:"Sexual activity: protected or unprotected. An empty value clears it." enum:"${menstrual_sexual_activity}," placeholder:"KIND"`
	Notes          *string  `help:"Notes. An empty value clears them." placeholder:"TEXT"`
	Ovulation      *bool    `help:"Mark the day as the ovulation day. --no-ovulation removes the mark." negatable:""`
}

func (c *WellnessMenstrualLogSetCmd) edit() garminapi.MenstrualLogEdit {
	return garminapi.MenstrualLogEdit{
		Flow:           garminMenstrualScalar("flow", c.Flow),
		Symptoms:       garminMenstrualList("symptoms", c.Symptoms),
		Moods:          garminMenstrualList("moods", c.Moods),
		Discharge:      garminMenstrualList("discharge", c.Discharge),
		SexDrive:       garminMenstrualScalar("sex_drive", c.SexDrive),
		SexualActivity: garminMenstrualScalar("sexual_activity", c.SexualActivity),
		Notes:          c.Notes,
		OvulationDay:   c.Ovulation,
	}
}

func garminMenstrualScalar(field string, v *string) *string {
	if v == nil {
		return nil
	}
	g := garminMenstrualValue(field, *v)
	return &g
}

func garminMenstrualList(field string, vs []string) *[]string {
	if vs == nil {
		return nil
	}
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, garminMenstrualValue(field, v))
	}
	return &out
}

func (c *WellnessMenstrualLogSetCmd) Validate() error {
	if c.edit() == (garminapi.MenstrualLogEdit{}) {
		return errors.New("nothing to set: give --flow, --symptoms, --moods, --discharge, --sex-drive, --sexual-activity, --notes, --ovulation or --no-ovulation")
	}
	if len(c.Discharge) > 1 && slices.Contains(c.Discharge, "no-discharge") {
		return errors.New("--discharge no-discharge takes no other value")
	}
	return nil
}

func (c *WellnessMenstrualLogSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.SetMenstrualLog(g.Context, string(c.Date), c.edit(), nowFn())
	if err != nil {
		return fmt.Errorf("set menstrual log: %w", err)
	}
	return reportWrite(g, data, "Saved menstrual log for %s", c.Date)
}

// WellnessMenstrualLogDeleteCmd deletes the menstrual daily log of a day.
type WellnessMenstrualLogDeleteCmd struct {
	Date  Date `arg:"" help:"Date of the log (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	Force bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *WellnessMenstrualLogDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	params := map[string]string{"date": string(c.Date)}
	// A DELETE of a day without a log also answers 204, so the read tells
	// whether there is a log to delete.
	saved, err := client.Call(g.Context, "wellness.menstrual-log", params, nil)
	if err != nil {
		return fmt.Errorf("get menstrual log: %w", err)
	}
	if saved == nil {
		return fmt.Errorf("no menstrual log on %s", c.Date)
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete menstrual log for %s?", c.Date), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err := client.Call(g.Context, "wellness.delete-menstrual-log", params, nil)
	if err != nil {
		return fmt.Errorf("delete menstrual log: %w", err)
	}
	return reportWrite(g, data, "Deleted menstrual log for %s", c.Date)
}
