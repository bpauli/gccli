package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var segmentsGroup = Group{Name: "segments", Help: "Find segments and see their leaderboards.", Children: []Node{
	Custom{Name: "list", Help: "List the segments you created, or your favorites with --favorites.", Default: true, Cmd: &SegmentsListCmd{}},
	Custom{Name: "search", Help: "Search segments by name or map area.", Cmd: &SegmentsSearchCmd{}},
	Leaf{
		Name: "show", Help: "Show a segment with its track points.",
		Spec: argsOf(&struct {
			UUID ID `arg:"" help:"Segment UUID."`
		}{}),
		Endpoint: "segments.get",
		Params:   Params{"uuid": In("UUID")},
		Verb:     "get segment", Output: JSON{},
	},
	Custom{Name: "leaderboard", Help: "Show the leaderboard of a segment.", Cmd: &SegmentsLeaderboardCmd{}},
	Leaf{
		Name: "rank", Help: "Show your best time and the top three on a segment.",
		Spec: argsOf(&struct {
			UUID ID `arg:"" help:"Segment UUID."`
		}{}),
		Endpoint: "segments.rank",
		Params:   Params{"uuid": In("UUID"), "displayName": DisplayName},
		Verb:     "get segment rank", Output: JSON{},
	},
	Leaf{Name: "classifications", Help: "List the segment classifications.", Endpoint: "segments.classifications", Verb: "get segment classifications", Output: JSON{List: true}},
	Leaf{Name: "surface-types", Help: "List the segment surface types.", Endpoint: "segments.surface-types", Verb: "get segment surface types", Output: JSON{List: true}},
}}

// The flag values of the segment kinds, mapped to Garmin's typeId. The
// enum tags of the flags list the same keys, which
// TestSegmentKindsMatchEnums checks. Garmin's "all" (typeId 1) means no
// filter, so it is not a value.
var (
	segmentActivityTypes   = map[string]int{"cycling": 2, "running": 1}
	segmentClassifications = map[string]int{"downhill": 12, "hill_climb": 8, "hills": 11, "other": 13, "sprint": 2}
	segmentSurfaces        = map[string]int{"dirt": 7, "gravel": 6, "mixed": 10, "paved": 2}
)

var segmentColumns = []Column{
	{Label: "UUID", Path: "uuid"},
	{Label: "NAME", Path: "name"},
	{Label: "DISTANCE", Path: "distance", Format: Kilometers},
	{Label: "FAVORITE", Path: "favorite"},
}

// segmentFilter is the start of a segments.search body. Garmin answers a
// filter without a keyword, a box, mineOnly or myFavorites with no
// segments.
func segmentFilter(page Pagination) map[string]any {
	return map[string]any{"showFavorites": true, "limit": page.Limit, "start": page.Offset}
}

func searchSegments(g *Globals, filter map[string]any, verb string) error {
	body, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "segments.search", nil, body)
	if err != nil {
		return fmt.Errorf("%s: %w", verb, err)
	}
	return Rows{Noun: "segments", Items: "segments", Columns: segmentColumns}.print(g, client, data)
}

// SegmentsListCmd lists the segments of the user.
type SegmentsListCmd struct {
	Favorites  bool `help:"List your favorite segments instead."`
	Pagination `embed:"" set:"items=segments"`
}

func (c *SegmentsListCmd) Run(g *Globals) error {
	filter := segmentFilter(c.Pagination)
	if c.Favorites {
		filter["myFavorites"] = true
	} else {
		filter["mineOnly"] = true
	}
	return searchSegments(g, filter, "list segments")
}

// SegmentsSearchCmd searches all segments by name or map area.
type SegmentsSearchCmd struct {
	Name           FreeText    `help:"Text in the segment name." placeholder:"TEXT"`
	Bbox           BoundingBox `help:"Map area as MINLAT,MINLON,MAXLAT,MAXLON in degrees." placeholder:"MINLAT,MINLON,MAXLAT,MAXLON"`
	ActivityType   *string     `help:"Activity type: cycling or running." enum:"cycling,running" placeholder:"TYPE"`
	Classification *string     `help:"Classification as segments classifications prints it: downhill, hill_climb, hills, other or sprint." enum:"downhill,hill_climb,hills,other,sprint" placeholder:"KEY"`
	Surface        *string     `help:"Surface as segments surface-types prints it: dirt, gravel, mixed or paved." enum:"dirt,gravel,mixed,paved" placeholder:"KEY"`
	MinDistance    *float64    `help:"Minimum distance in kilometers." placeholder:"KM"`
	MaxDistance    *float64    `help:"Maximum distance in kilometers." placeholder:"KM"`
	MinGrade       *float64    `help:"Minimum average grade in percent." placeholder:"PCT"`
	MaxGrade       *float64    `help:"Maximum average grade in percent." placeholder:"PCT"`
	Pagination     `embed:"" set:"items=segments"`
}

func (c *SegmentsSearchCmd) Validate() error {
	if err := c.Pagination.Validate(); err != nil {
		return err
	}
	if c.Name == "" && c.Bbox == (BoundingBox{}) {
		return errors.New("give --name or --bbox: Garmin finds no segments without one")
	}
	return nil
}

func (c *SegmentsSearchCmd) filter() map[string]any {
	f := segmentFilter(c.Pagination)
	f["mineOnly"] = false
	if c.Name != "" {
		f["keyword"] = string(c.Name)
	}
	if c.Bbox != (BoundingBox{}) {
		f["minLat"], f["minLon"], f["maxLat"], f["maxLon"] = c.Bbox.MinLat, c.Bbox.MinLon, c.Bbox.MaxLat, c.Bbox.MaxLon
	}
	// Garmin wants these IDs and limits as strings: a number array is a 500.
	kinds := []struct {
		key  string
		flag *string
		ids  map[string]int
	}{
		{"activityTypePk", c.ActivityType, segmentActivityTypes},
		{"classificationPk", c.Classification, segmentClassifications},
		{"surfaceTypePk", c.Surface, segmentSurfaces},
	}
	for _, k := range kinds {
		if k.flag != nil {
			f[k.key] = strconv.Itoa(k.ids[*k.flag])
		}
	}
	limits := []struct {
		key  string
		flag *float64
	}{
		{"minDistance", c.MinDistance},
		{"maxDistance", c.MaxDistance},
		{"minGrade", c.MinGrade},
		{"maxGrade", c.MaxGrade},
	}
	for _, l := range limits {
		if l.flag != nil {
			f[l.key] = strconv.FormatFloat(*l.flag, 'f', -1, 64)
		}
	}
	return f
}

func (c *SegmentsSearchCmd) Run(g *Globals) error {
	return searchSegments(g, c.filter(), "search segments")
}

// leaderboardScopes maps each --scope to the body key that limits the
// leaderboard, or "" for none.
var leaderboardScopes = map[string]string{"all": "", "connections": "connectionsOnly", "following": "followingsOnly", "me": "meOnly"}

var leaderboardColumns = []Column{
	{Label: "RANK", Path: "rank"},
	{Label: "NAME", Path: "fullName"},
	{Label: "TIME", Path: "elapsedMilliseconds", Format: Milliseconds},
	{Label: "DATE", Path: "timeEntered", Format: DateOnly},
	{Label: "ACTIVITY", Path: "activityID"},
}

// SegmentsLeaderboardCmd shows one page of a segment leaderboard.
type SegmentsLeaderboardCmd struct {
	UUID     ID        `arg:"" help:"Segment UUID."`
	FromRank int       `help:"Rank to start at. One page has 26 rows." default:"1"`
	Scope    string    `help:"Whose times: all, connections, following or me." default:"all" enum:"all,connections,following,me"`
	Group    NumericID `help:"Show only the members of this group. Goes only with --scope all." placeholder:"ID"`
	Gender   *string   `help:"Gender: female or male." enum:"female,male" placeholder:"GENDER"`
	Age      *string   `help:"Age group: 0-18, 18-24, 25-34, 35-44, 45-54, 55-64 or 65-120." enum:"0-18,18-24,25-34,35-44,45-54,55-64,65-120" placeholder:"AGES"`
}

func (c *SegmentsLeaderboardCmd) Validate() error {
	if c.FromRank < 1 {
		return fmt.Errorf("--from-rank must be at least 1, got %d", c.FromRank)
	}
	if c.Group != 0 && c.Scope != "all" {
		return fmt.Errorf("--group cannot go with --scope %s: a group leaderboard has the times of all its members", c.Scope)
	}
	return nil
}

func (c *SegmentsLeaderboardCmd) filter() map[string]any {
	f := map[string]any{"segmentUuid": string(c.UUID)}
	if c.FromRank > 1 {
		f["startingRowNumber"] = c.FromRank
	}
	if key := leaderboardScopes[c.Scope]; key != "" {
		f[key] = "true"
	}
	if c.Group != 0 {
		f["groupPk"] = int64(c.Group)
	}
	if c.Gender != nil {
		f["gender"] = *c.Gender
	}
	if c.Age != nil {
		f["ageMin"], f["ageMax"], _ = strings.Cut(*c.Age, "-")
	}
	return f
}

func (c *SegmentsLeaderboardCmd) Run(g *Globals) error {
	body, err := json.Marshal(c.filter())
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "segments.leaderboard", nil, body)
	if err != nil {
		return fmt.Errorf("get segment leaderboard: %w", err)
	}
	return Rows{Noun: "leaderboard", Items: "leaderboardRowDTOList", Columns: leaderboardColumns}.print(g, client, data)
}
