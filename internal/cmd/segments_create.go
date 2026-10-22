package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
)

// SegmentsCreateCmd creates a segment from a part of an activity track.
type SegmentsCreateCmd struct {
	FromActivity   NumericID `help:"ID of the activity whose track the segment follows." required:"" placeholder:"ID"`
	StartKm        float64   `help:"Start of the segment, in kilometers from the start of the activity." required:"" placeholder:"KM"`
	EndKm          float64   `help:"End of the segment, in kilometers from the start of the activity." required:"" placeholder:"KM"`
	Name           FreeText  `help:"Segment name." required:"" placeholder:"NAME"`
	ActivityType   string    `help:"Activity type: cycling or running." required:"" enum:"cycling,running" placeholder:"TYPE"`
	Classification string    `help:"Classification as segments classifications prints it: downhill, hill_climb, hills, other or sprint." default:"other" enum:"downhill,hill_climb,hills,other,sprint"`
	Surface        string    `help:"Surface as segments surface-types prints it: dirt, gravel, mixed or paved." default:"paved" enum:"dirt,gravel,mixed,paved"`
}

func (c *SegmentsCreateCmd) Validate() error {
	// Written so that NaN fails it.
	if !(c.StartKm >= 0 && c.StartKm < c.EndKm) {
		return fmt.Errorf("--start-km must be at least 0 and less than --end-km, got %s and %s", km(c.StartKm), km(c.EndKm))
	}
	return nil
}

func km(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// trackPoint is one point of an activity track. raw is the point as Garmin
// sent it: the new segment carries it unchanged.
type trackPoint struct {
	raw json.RawMessage
	// distance is in meters from the start of the activity.
	distance  float64
	elevation *float64
}

func parseTrack(data json.RawMessage) ([]trackPoint, error) {
	var track struct {
		Polyline []json.RawMessage `json:"polyline"`
	}
	if err := unmarshalAnswer(data, &track); err != nil {
		return nil, err
	}
	points := make([]trackPoint, 0, len(track.Polyline))
	for _, raw := range track.Polyline {
		var p struct {
			Distance  float64  `json:"distance"`
			Elevation *float64 `json:"elevation"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		points = append(points, trackPoint{raw: raw, distance: p.Distance, elevation: p.Elevation})
	}
	return points, nil
}

// cutTrack returns the points from the first at or after startM to the last
// at or before endM, or nil when there are fewer than two.
func cutTrack(points []trackPoint, startM, endM float64) []trackPoint {
	first := slices.IndexFunc(points, func(p trackPoint) bool { return p.distance >= startM })
	last := len(points) - 1
	for last >= 0 && points[last].distance > endM {
		last--
	}
	if first < 0 || last-first < 1 {
		return nil
	}
	return points[first : last+1]
}

// segmentGrade is the average grade from the first to the last point in
// percent, rounded to 2 decimals, or 0 without both elevations or without a
// positive run.
func segmentGrade(span []trackPoint) float64 {
	start, end := span[0], span[len(span)-1]
	run := end.distance - start.distance
	if start.elevation == nil || end.elevation == nil || run <= 0 {
		return 0
	}
	return math.Round(100*(*end.elevation-*start.elevation)/run*100) / 100
}

// segmentDraft is the body of segments.create, as the Garmin Connect web
// app sends it.
type segmentDraft struct {
	Name             string            `json:"name"`
	ActivityID       NumericID         `json:"activityId"`
	ActivityTypePk   int               `json:"activityTypePk"`
	ClassificationPk int               `json:"classificationPk"`
	SurfaceTypePk    int               `json:"surfaceTypePk"`
	Distance         float64           `json:"distance"`
	Grade            float64           `json:"grade"`
	GeoPoints        []json.RawMessage `json:"geoPoints"`
}

func (c *SegmentsCreateCmd) draft(span []trackPoint) segmentDraft {
	geo := make([]json.RawMessage, len(span))
	for i, p := range span {
		geo[i] = p.raw
	}
	return segmentDraft{
		Name:             string(c.Name),
		ActivityID:       c.FromActivity,
		ActivityTypePk:   segmentActivityTypes[c.ActivityType],
		ClassificationPk: segmentClassifications[c.Classification],
		SurfaceTypePk:    segmentSurfaces[c.Surface],
		Distance:         (span[len(span)-1].distance - span[0].distance) / 1000,
		Grade:            segmentGrade(span),
		GeoPoints:        geo,
	}
}

func (c *SegmentsCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "segments.track", map[string]string{"activityID": c.FromActivity.String()}, nil)
	if err != nil {
		return fmt.Errorf("get activity track: %w", err)
	}
	points, err := parseTrack(data)
	if err != nil {
		return fmt.Errorf("parse activity track: %w", err)
	}
	span := cutTrack(points, c.StartKm*1000, c.EndKm*1000)
	if span == nil {
		length := 0.0
		if len(points) > 0 {
			length = points[len(points)-1].distance / 1000
		}
		return fmt.Errorf("the track of activity %s has fewer than 2 points from %s km to %s km: the track is %.2f km long", c.FromActivity, km(c.StartKm), km(c.EndKm), length)
	}
	body, err := json.Marshal(c.draft(span))
	if err != nil {
		return err
	}
	saved, err := client.Call(g.Context, "segments.create", nil, body)
	if err != nil {
		return fmt.Errorf("create segment: %w", err)
	}
	var created struct {
		UUID string `json:"uuid"`
	}
	_ = json.Unmarshal(saved, &created)
	if created.UUID != "" {
		return reportWrite(g, saved, "Created segment %q (UUID: %s)", c.Name, created.UUID)
	}
	return reportWrite(g, saved, "Created segment %q", c.Name)
}
