package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// JetLagFlight is one flight of a jet lag trip. Depart and Arrive are the
// local clock times at the two places, and their zone offsets are the UTC
// offsets there, because Garmin keeps each time with its own offset.
type JetLagFlight struct {
	From, To       string
	Depart, Arrive time.Time
}

// JetLagTrip is a saved jet lag trip. Raw is the trip as Garmin sent it.
type JetLagTrip struct {
	Raw                json.RawMessage
	StartDate, EndDate string
	Flights            []JetLagFlight
}

// JetLagTripEdit holds the changes to a trip of one flight, or of a flight
// and its return flight. An empty string or a zero time keeps the saved
// value. From and To name the places of the first flight, and the return
// flight takes them the other way round.
type JetLagTripEdit struct {
	From, To                   string
	Depart, Arrive             time.Time
	ReturnDepart, ReturnArrive time.Time
}

// jetLagTimestamp is the local time format of the flight timestamps. The
// UTC offset is in a field of its own.
const jetLagTimestamp = "2006-01-02T15:04:05.0"

type jetLagFlightJSON struct {
	OriginLocation       string `json:"originLocation"`
	OriginTimestamp      string `json:"originTimestamp"`
	OriginTimezone       string `json:"originTimezone"`
	DestinationLocation  string `json:"destinationLocation"`
	DestinationTimestamp string `json:"destinationTimestamp"`
	DestinationTimezone  string `json:"destinationTimezone"`
}

func (f JetLagFlight) fields() map[string]any {
	return map[string]any{
		"originLocation":       f.From,
		"originTimestamp":      f.Depart.Format(jetLagTimestamp),
		"originTimezone":       f.Depart.Format("-07:00"),
		"destinationLocation":  f.To,
		"destinationTimestamp": f.Arrive.Format(jetLagTimestamp),
		"destinationTimezone":  f.Arrive.Format("-07:00"),
	}
}

func (w jetLagFlightJSON) flight() (JetLagFlight, error) {
	depart, err := time.Parse(time.RFC3339, w.OriginTimestamp+w.OriginTimezone)
	if err != nil {
		return JetLagFlight{}, errors.New("a flight has no valid departure time")
	}
	arrive, err := time.Parse(time.RFC3339, w.DestinationTimestamp+w.DestinationTimezone)
	if err != nil {
		return JetLagFlight{}, errors.New("a flight has no valid arrival time")
	}
	return JetLagFlight{From: w.OriginLocation, To: w.DestinationLocation, Depart: depart, Arrive: arrive}, nil
}

// ValidateJetLagFlights checks that each flight arrives after it departs and
// that each flight departs after the one before it arrives.
func ValidateJetLagFlights(flights []JetLagFlight) error {
	if len(flights) == 0 {
		return errors.New("a trip needs at least one flight")
	}
	for i, f := range flights {
		if !f.Arrive.After(f.Depart) {
			return fmt.Errorf("the arrival %s must be after the departure %s", stamp(f.Arrive), stamp(f.Depart))
		}
		if i > 0 && !f.Depart.After(flights[i-1].Arrive) {
			return fmt.Errorf("the return departure %s must be after the arrival %s", stamp(f.Depart), stamp(flights[i-1].Arrive))
		}
	}
	return nil
}

func stamp(t time.Time) string { return t.Format("2006-01-02T15:04-07:00") }

// tripDates returns the local day of the first departure and of the last
// arrival. Garmin rejects a trip whose flights are not inside its dates.
func tripDates(flights []JetLagFlight) (start, end string) {
	return flights[0].Depart.Format(time.DateOnly), flights[len(flights)-1].Arrive.Format(time.DateOnly)
}

// AddJetLagTrip saves a trip of the flights. The trip runs from the day of
// the first departure to the day of the last arrival.
func (c *Client) AddJetLagTrip(ctx context.Context, flights []JetLagFlight) (json.RawMessage, error) {
	wire := make([]map[string]any, len(flights))
	for i, f := range flights {
		wire[i] = f.fields()
	}
	start, end := tripDates(flights)
	body, err := json.Marshal(map[string]any{"startDate": start, "endDate": end, "flights": wire})
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "wellness.add-jet-lag-trip", nil, body)
}

// JetLagTrip returns the trip with the ID from the trip list, because
// Garmin has no request for one trip.
func (c *Client) JetLagTrip(ctx context.Context, id string) (JetLagTrip, error) {
	data, err := c.Call(ctx, "wellness.jet-lag-trips", nil, nil)
	if err != nil {
		return JetLagTrip{}, err
	}
	var trips []json.RawMessage
	if len(data) > 0 {
		if err := json.Unmarshal(data, &trips); err != nil {
			return JetLagTrip{}, errors.New("parse jet lag trips: not a JSON array")
		}
	}
	for _, raw := range trips {
		var t struct {
			TripID    json.Number        `json:"tripId"`
			StartDate string             `json:"startDate"`
			EndDate   string             `json:"endDate"`
			Flights   []jetLagFlightJSON `json:"flights"`
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			return JetLagTrip{}, fmt.Errorf("parse jet lag trips: %w", err)
		}
		if t.TripID.String() != id {
			continue
		}
		trip := JetLagTrip{Raw: raw, StartDate: t.StartDate, EndDate: t.EndDate}
		for _, w := range t.Flights {
			f, err := w.flight()
			if err != nil {
				return JetLagTrip{}, fmt.Errorf("parse jet lag trip %s: %w", id, err)
			}
			trip.Flights = append(trip.Flights, f)
		}
		return trip, nil
	}
	return JetLagTrip{}, fmt.Errorf("no jet lag trip has the ID %s", id)
}

// UpdateJetLagTrip reads the trip, applies e and saves the full trip, with
// its dates set again from the flights. Fields that e does not know, such as
// the flight IDs, are sent back as Garmin sent them.
func (c *Client) UpdateJetLagTrip(ctx context.Context, id string, e JetLagTripEdit) (json.RawMessage, error) {
	trip, err := c.JetLagTrip(ctx, id)
	if err != nil {
		return nil, err
	}
	flights, err := e.apply(trip.Flights)
	if err != nil {
		return nil, err
	}
	body, err := jetLagTripBody(trip.Raw, flights)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "wellness.update-jet-lag-trip", nil, body)
}

func (e JetLagTripEdit) apply(saved []JetLagFlight) ([]JetLagFlight, error) {
	if len(saved) == 0 {
		return nil, errors.New("the trip has no flights")
	}
	if len(saved) > 2 {
		return nil, fmt.Errorf("the trip has %d flights, and gccli changes only a trip of one flight or of a flight and its return flight", len(saved))
	}
	hasReturn := len(saved) == 2
	if !hasReturn && (!e.ReturnDepart.IsZero() || !e.ReturnArrive.IsZero()) {
		return nil, errors.New("the trip has no return flight")
	}
	flights := append([]JetLagFlight(nil), saved...)
	out := &flights[0]
	setString(&out.From, e.From)
	setString(&out.To, e.To)
	setTime(&out.Depart, e.Depart)
	setTime(&out.Arrive, e.Arrive)
	if hasReturn {
		back := &flights[1]
		setString(&back.From, e.To)
		setString(&back.To, e.From)
		setTime(&back.Depart, e.ReturnDepart)
		setTime(&back.Arrive, e.ReturnArrive)
	}
	if err := ValidateJetLagFlights(flights); err != nil {
		return nil, err
	}
	return flights, nil
}

func setString(dst *string, v string) {
	if v != "" {
		*dst = v
	}
}

func setTime(dst *time.Time, v time.Time) {
	if !v.IsZero() {
		*dst = v
	}
}

// jetLagTripBody returns the saved trip with the flights and the dates set.
func jetLagTripBody(saved json.RawMessage, flights []JetLagFlight) ([]byte, error) {
	var trip struct {
		Flights []json.RawMessage `json:"flights"`
	}
	if err := json.Unmarshal(saved, &trip); err != nil || len(trip.Flights) != len(flights) {
		return nil, errors.New("parse jet lag trip: the flights do not match")
	}
	merged := make([]json.RawMessage, len(flights))
	for i, f := range flights {
		body, err := setFields(trip.Flights[i], "flight", f.fields())
		if err != nil {
			return nil, err
		}
		merged[i] = body
	}
	start, end := tripDates(flights)
	return setFields(saved, "trip", map[string]any{"startDate": start, "endDate": end, "flights": merged})
}
