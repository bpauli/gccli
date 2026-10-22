package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bpauli/gccli/internal/garminapi"
)

var wellnessJetLagGroup = Group{Name: "jet-lag", Help: "Plan jet lag trips and show jet lag status.", Children: []Node{
	Leaf{Name: "list", Help: "List jet lag trips.", Default: true, Endpoint: "wellness.jet-lag-trips", Verb: "get jet lag trips", Output: JSON{List: true}},
	Custom{Name: "add", Help: "Add a jet lag trip of a flight and an optional return flight.", Cmd: &WellnessJetLagAddCmd{}},
	Custom{Name: "update", Help: "Change the places or times of a jet lag trip.", Cmd: &WellnessJetLagUpdateCmd{}},
	Custom{Name: "delete", Help: "Delete a jet lag trip.", Cmd: &WellnessJetLagDeleteCmd{}},
	Leaf{
		Name: "status", Help: "Show jet lag status for a date.",
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "wellness.jet-lag-status",
		Params:   Params{"date": In("Date")},
		Verb:     "get jet lag status", Output: JSON{},
	},
}}

// WellnessJetLagAddCmd adds a jet lag trip.
type WellnessJetLagAddCmd struct {
	From         FreeText   `required:"" help:"Place of departure, such as \"Frankfurt, Germany\"." placeholder:"PLACE"`
	To           FreeText   `required:"" help:"Place of arrival, such as \"New York, USA\"." placeholder:"PLACE"`
	Depart       FlightTime `required:"" placeholder:"TIME" help:"Local departure time with the UTC offset of --from, such as 2026-11-02T10:00+01:00."`
	Arrive       FlightTime `required:"" placeholder:"TIME" help:"Local arrival time with the UTC offset of --to, such as 2026-11-02T13:00-05:00."`
	ReturnDepart FlightTime `placeholder:"TIME" help:"Local departure time of the return flight from --to, with its UTC offset."`
	ReturnArrive FlightTime `placeholder:"TIME" help:"Local arrival time of the return flight at --from, with its UTC offset."`
}

func (c *WellnessJetLagAddCmd) flights() []garminapi.JetLagFlight {
	flights := []garminapi.JetLagFlight{{From: string(c.From), To: string(c.To), Depart: c.Depart.Time, Arrive: c.Arrive.Time}}
	if !c.ReturnDepart.IsZero() {
		flights = append(flights, garminapi.JetLagFlight{From: string(c.To), To: string(c.From), Depart: c.ReturnDepart.Time, Arrive: c.ReturnArrive.Time})
	}
	return flights
}

func (c *WellnessJetLagAddCmd) Validate() error {
	if c.ReturnDepart.IsZero() != c.ReturnArrive.IsZero() {
		return errors.New("a return flight needs both --return-depart and --return-arrive")
	}
	return garminapi.ValidateJetLagFlights(c.flights())
}

func (c *WellnessJetLagAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.AddJetLagTrip(g.Context, c.flights())
	if err != nil {
		return fmt.Errorf("add jet lag trip: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "tripId"); id != "" {
		return reportWrite(g, data, "Added jet lag trip from %s to %s (ID: %s)", c.From, c.To, id)
	}
	return reportWrite(g, data, "Added jet lag trip from %s to %s", c.From, c.To)
}

// WellnessJetLagUpdateCmd changes a jet lag trip of one flight, or of a
// flight and its return flight.
type WellnessJetLagUpdateCmd struct {
	ID           ID         `arg:"" help:"Trip ID."`
	From         FreeText   `help:"New place of departure. The return flight arrives there." placeholder:"PLACE"`
	To           FreeText   `help:"New place of arrival. The return flight departs there." placeholder:"PLACE"`
	Depart       FlightTime `placeholder:"TIME" help:"New local departure time with the UTC offset, such as 2026-11-02T10:00+01:00."`
	Arrive       FlightTime `placeholder:"TIME" help:"New local arrival time with the UTC offset."`
	ReturnDepart FlightTime `placeholder:"TIME" help:"New local departure time of the return flight, with the UTC offset."`
	ReturnArrive FlightTime `placeholder:"TIME" help:"New local arrival time of the return flight, with the UTC offset."`
}

func (c *WellnessJetLagUpdateCmd) edit() garminapi.JetLagTripEdit {
	return garminapi.JetLagTripEdit{
		From: string(c.From), To: string(c.To),
		Depart: c.Depart.Time, Arrive: c.Arrive.Time,
		ReturnDepart: c.ReturnDepart.Time, ReturnArrive: c.ReturnArrive.Time,
	}
}

func (c *WellnessJetLagUpdateCmd) Validate() error {
	if c.edit() == (garminapi.JetLagTripEdit{}) {
		return errors.New("nothing to change: give --from, --to, --depart, --arrive, --return-depart or --return-arrive")
	}
	return nil
}

func (c *WellnessJetLagUpdateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateJetLagTrip(g.Context, string(c.ID), c.edit())
	if err != nil {
		return fmt.Errorf("update jet lag trip: %w", err)
	}
	return reportWrite(g, data, "Updated jet lag trip %s", c.ID)
}

// WellnessJetLagDeleteCmd deletes a jet lag trip. A DELETE of a trip that
// does not exist also answers 204, so the trip comes from the trip list
// first: a missing trip is an error.
type WellnessJetLagDeleteCmd struct {
	ID    ID   `arg:"" help:"Trip ID."`
	Force bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *WellnessJetLagDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	trip, err := client.JetLagTrip(g.Context, string(c.ID))
	if err != nil {
		return fmt.Errorf("get jet lag trip: %w", err)
	}
	prompt := fmt.Sprintf("Delete jet lag trip %s from %s to %s?", c.ID, trip.StartDate, trip.EndDate)
	if len(trip.Flights) > 0 {
		prompt = fmt.Sprintf("Delete jet lag trip %s (%s to %s, %s to %s)?", c.ID, trip.Flights[0].From, trip.Flights[0].To, trip.StartDate, trip.EndDate)
	}
	ok, err := confirm(g.Stderr, prompt, c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err := client.Call(g.Context, "wellness.delete-jet-lag-trip", map[string]string{"tripID": string(c.ID)}, nil)
	if err != nil {
		return fmt.Errorf("delete jet lag trip: %w", err)
	}
	return reportWrite(g, data, "Deleted jet lag trip %s", c.ID)
}
