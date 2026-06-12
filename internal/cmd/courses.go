package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
)

// CoursesCmd groups course subcommands.
type CoursesCmd struct {
	List      CoursesListCmd      `cmd:"" default:"withargs" help:"List courses."`
	Favorites CoursesFavoritesCmd `cmd:"" help:"List favorite courses."`
	Detail    CourseDetailCmd     `cmd:"" help:"View course details."`
	Download  CourseDownloadCmd   `cmd:"" help:"Download course file."`
	Import    CourseImportCmd     `cmd:"" help:"Import a GPX file as a new course."`
	Send      CourseSendCmd       `cmd:"" help:"Send a course to a device."`
	Delete    CourseDeleteCmd     `cmd:"" help:"Delete a course."`
}

// CoursesListCmd lists the user's courses.
type CoursesListCmd struct{}

func (c *CoursesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetCourses(g.Context)
	if err != nil {
		return fmt.Errorf("list courses: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	courses, err := parseCourses(data)
	if err != nil {
		return err
	}

	rows := formatCourseRows(courses)
	header := []string{"ID", "NAME", "TYPE", "DISTANCE", "CREATED"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
}

// CoursesFavoritesCmd lists favorite courses.
type CoursesFavoritesCmd struct{}

func (c *CoursesFavoritesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetCourseFavorites(g.Context)
	if err != nil {
		return fmt.Errorf("list favorite courses: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	var courses []map[string]any
	if err := json.Unmarshal(data, &courses); err != nil {
		return fmt.Errorf("parse favorite courses: %w", err)
	}

	rows := formatCourseRows(courses)
	header := []string{"ID", "NAME", "TYPE", "DISTANCE", "CREATED"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
}

// CourseDetailCmd shows details for a course.
type CourseDetailCmd struct {
	ID string `arg:"" help:"Course ID."`
}

func (c *CourseDetailCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetCourse(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get course: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
}

// CourseDownloadCmd downloads a course in a specified format.
type CourseDownloadCmd struct {
	ID     string `arg:"" help:"Course ID."`
	Format string `help:"Download format: fit, gpx." default:"fit" enum:"fit,gpx"`
	Output string `help:"Output file path (default: course_{id}.{format})." short:"o"`
}

func (c *CourseDownloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	format := garminapi.ActivityDownloadFormat(strings.ToLower(c.Format))

	data, err := client.DownloadCourse(g.Context, c.ID, format)
	if err != nil {
		return fmt.Errorf("download course: %w", err)
	}

	outPath := c.Output
	if outPath == "" {
		outPath = fmt.Sprintf("course_%s.%s", c.ID, c.Format)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	g.UI.Successf("Downloaded %s (%d bytes)", outPath, len(data))
	return nil
}

// CourseImportCmd imports a GPX file as a new course.
type CourseImportCmd struct {
	File    string `arg:"" help:"Path to GPX file." type:"existingfile"`
	Name    string `help:"Course name (overrides name from GPX)." short:"n"`
	Type    string `help:"Activity type key (e.g. running, cycling, hiking)." short:"t" default:"cycling"`
	Privacy int    `help:"Course privacy: 1=public, 2=private, 4=group." short:"p" default:"2" enum:"1,2,4"`
}

func (c *CourseImportCmd) Run(g *Globals) error {
	typePk, err := resolveCourseActivityType(c.Type)
	if err != nil {
		return err
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	// Step 1: Parse GPX.
	g.UI.Infof("Importing %s...", filepath.Base(c.File))
	parsed, err := client.ImportCourseGPX(g.Context, c.File)
	if err != nil {
		return fmt.Errorf("import GPX: %w", err)
	}

	var course map[string]any
	if err := json.Unmarshal(parsed, &course); err != nil {
		return fmt.Errorf("parse imported course: %w", err)
	}

	// Step 2: Enrich elevation.
	geoPoints, ok := course["geoPoints"].([]any)
	if !ok || len(geoPoints) == 0 {
		return fmt.Errorf("imported course has no geo points")
	}

	elevInput := make([][]any, 0, len(geoPoints))
	for _, p := range geoPoints {
		pt, ok := p.(map[string]any)
		if !ok {
			continue
		}
		lat := pt["latitude"]
		lon := pt["longitude"]
		if lat == nil || lon == nil {
			lat = pt["lat"]
			lon = pt["lon"]
		}
		if lat == nil || lon == nil {
			continue
		}
		elevInput = append(elevInput, []any{lat, lon, nil})
	}

	elevJSON, err := json.Marshal(elevInput)
	if err != nil {
		return fmt.Errorf("marshal elevation input: %w", err)
	}

	elevData, err := client.GetCourseElevation(g.Context, elevJSON)
	if err != nil {
		return fmt.Errorf("get elevation: %w", err)
	}

	var elevPoints [][]any
	if err := json.Unmarshal(elevData, &elevPoints); err != nil {
		return fmt.Errorf("parse elevation: %w", err)
	}

	// Merge elevation back into geoPoints.
	for i, ep := range elevPoints {
		if i >= len(geoPoints) {
			break
		}
		if pt, ok := geoPoints[i].(map[string]any); ok && len(ep) >= 3 {
			pt["elevation"] = ep[2]
		}
	}
	course["geoPoints"] = geoPoints

	// Override course name.
	if c.Name != "" {
		course["courseName"] = c.Name
	} else if course["courseName"] == nil || jsonString(course, "courseName") == "" {
		// Fallback to filename without extension.
		course["courseName"] = strings.TrimSuffix(filepath.Base(c.File), filepath.Ext(c.File))
	}

	// Set activity type and defaults for fields the import response leaves nil.
	course["activityTypePk"] = typePk
	if course["coordinateSystem"] == nil {
		course["coordinateSystem"] = "WGS84"
	}
	if course["sourceTypeId"] == nil {
		course["sourceTypeId"] = 3
	}
	if course["startPoint"] == nil && len(geoPoints) > 0 {
		course["startPoint"] = geoPoints[0]
	}

	// Build save payload with only the fields the API accepts.
	save := filterCourseFields(course)
	save["coursePrivacy"] = c.Privacy
	if save["rulePK"] == nil {
		save["rulePK"] = 2
	}

	payload, err := json.Marshal(save)
	if err != nil {
		return fmt.Errorf("marshal course: %w", err)
	}

	// Step 3: Save course.
	saved, err := client.SaveCourse(g.Context, payload)
	if err != nil {
		return fmt.Errorf("save course: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(saved))
	}

	var result map[string]any
	if err := json.Unmarshal(saved, &result); err != nil {
		return fmt.Errorf("parse saved course: %w", err)
	}

	rows := [][]string{{
		jsonString(result, "courseId"),
		jsonString(result, "courseName"),
		formatDistance(jsonFloat(result, "distanceMeter")),
		formatElevation(jsonFloat(result, "elevationGainMeter")),
		formatElevation(jsonFloat(result, "elevationLossMeter")),
	}}
	header := []string{"ID", "NAME", "DISTANCE", "ELEV GAIN", "ELEV LOSS"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
}

// CourseSendCmd sends a course to a device.
type CourseSendCmd struct {
	CourseID string `arg:"" help:"Course ID."`
	DeviceID string `arg:"" help:"Device ID."`
}

func (c *CourseSendCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	// Fetch course detail to get the course name.
	courseData, err := client.GetCourse(g.Context, c.CourseID)
	if err != nil {
		return fmt.Errorf("get course: %w", err)
	}

	var course map[string]any
	if err := json.Unmarshal(courseData, &course); err != nil {
		return fmt.Errorf("parse course: %w", err)
	}
	courseName := jsonString(course, "courseName")

	data, err := client.SendCourseToDevice(g.Context, c.CourseID, c.DeviceID, courseName)
	if err != nil {
		return fmt.Errorf("send course to device: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	// Extract response details for success message.
	var msgs []map[string]any
	if err := json.Unmarshal(data, &msgs); err == nil && len(msgs) > 0 {
		deviceName := jsonString(msgs[0], "deviceName")
		messageID := jsonString(msgs[0], "messageId")
		g.UI.Successf("Sent course %q to device %s (message %s)", courseName, deviceName, messageID)
	} else {
		g.UI.Successf("Sent course %q to device %s", courseName, c.DeviceID)
	}

	return nil
}

// CourseDeleteCmd deletes a course.
type CourseDeleteCmd struct {
	ID    string `arg:"" help:"Course ID."`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *CourseDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete course %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteCourse(g.Context, c.ID); err != nil {
		return fmt.Errorf("delete course: %w", err)
	}

	g.UI.Successf("Deleted course %s", c.ID)
	return nil
}

// parseCourses extracts the course list from the API response.
// The list endpoint returns {coursesForUser: [...]}.
func parseCourses(data json.RawMessage) ([]map[string]any, error) {
	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse courses: %w", err)
	}

	arr, ok := wrapper["coursesForUser"]
	if !ok {
		return nil, nil
	}

	items, ok := arr.([]any)
	if !ok {
		return nil, nil
	}

	courses := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			courses = append(courses, m)
		}
	}
	return courses, nil
}

// formatCourseRows extracts table rows from course data.
func formatCourseRows(courses []map[string]any) [][]string {
	rows := make([][]string, 0, len(courses))
	for _, c := range courses {
		rows = append(rows, []string{
			jsonString(c, "courseId"),
			jsonString(c, "courseName"),
			courseTypeKey(c),
			formatDistance(jsonFloat(c, "distanceInMeters")),
			formatDate(jsonString(c, "createdDate")),
		})
	}
	return rows
}

// courseSaveFields are the fields accepted by the course save endpoint.
var courseSaveFields = []string{
	"activityTypePk", "boundingBox", "coordinateSystem", "courseLines",
	"courseName", "coursePoints", "distanceMeter", "elapsedSeconds",
	"elevationGainMeter", "elevationLossMeter", "favorite", "geoPoints",
	"hasPaceBand", "hasPowerGuide", "hasTurnDetectionDisabled", "includeLaps",
	"matchedToSegments", "openStreetMap", "rulePK", "sourceTypeId",
	"speedMeterPerSecond", "startPoint", "userProfilePk",
}

// filterCourseFields returns a new map containing only the non-nil fields
// accepted by the course save endpoint, dropping extra fields from the import response.
func filterCourseFields(course map[string]any) map[string]any {
	save := make(map[string]any, len(courseSaveFields))
	for _, key := range courseSaveFields {
		if v, ok := course[key]; ok && v != nil {
			save[key] = v
		}
	}
	return save
}

// courseTypeKey extracts the course type from nested courseType.typeKey.
func courseTypeKey(c map[string]any) string {
	ct, ok := c["courseType"]
	if !ok || ct == nil {
		return ""
	}
	if m, ok := ct.(map[string]any); ok {
		return jsonString(m, "typeKey")
	}
	return ""
}
