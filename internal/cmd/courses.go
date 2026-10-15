package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/units"
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
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	courses, err := parseCourses(data)
	if err != nil {
		return err
	}

	rows := formatCourseRows(resolveUnits(g, client), courses)
	header := []string{"ID", "NAME", "TYPE", "DISTANCE", "CREATED"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(g.Stdout, rows)
	}
	return outfmt.WriteTable(g.Stdout, header, rows)
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
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	var courses []map[string]any
	if err := json.Unmarshal(data, &courses); err != nil {
		return fmt.Errorf("parse favorite courses: %w", err)
	}

	rows := formatCourseRows(resolveUnits(g, client), courses)
	header := []string{"ID", "NAME", "TYPE", "DISTANCE", "CREATED"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(g.Stdout, rows)
	}
	return outfmt.WriteTable(g.Stdout, header, rows)
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

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
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

	format := garminapi.DownloadFormat(strings.ToLower(c.Format))

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

	g.UI.Infof("Importing %s...", filepath.Base(c.File))
	saved, err := client.ImportCourse(g.Context, c.File, garminapi.CourseImportOptions{
		Name:           c.Name,
		ActivityTypePK: typePk,
		Privacy:        c.Privacy,
	})
	if err != nil {
		return err
	}

	header := []string{"ID", "NAME", "DISTANCE", "ELEV GAIN", "ELEV LOSS"}
	return reportWriteTable(g, saved, "Imported course from "+filepath.Base(c.File), header, func() ([][]string, error) {
		var result map[string]any
		if err := json.Unmarshal(saved, &result); err != nil {
			return nil, fmt.Errorf("parse saved course: %w", err)
		}
		sys := resolveUnits(g, client)
		return [][]string{{
			jsonString(result, "courseId"),
			jsonString(result, "courseName"),
			sys.Distance(jsonFloat(result, "distanceMeter")),
			sys.Elevation(jsonFloat(result, "elevationGainMeter")),
			sys.Elevation(jsonFloat(result, "elevationLossMeter")),
		}}, nil
	})
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

	var msgs []map[string]any
	if err := json.Unmarshal(data, &msgs); err == nil && len(msgs) > 0 {
		deviceName := jsonString(msgs[0], "deviceName")
		messageID := jsonString(msgs[0], "messageId")
		return reportWrite(g, data, "Sent course %q to device %s (message %s)", courseName, deviceName, messageID)
	}
	return reportWrite(g, data, "Sent course %q to device %s", courseName, c.DeviceID)
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

	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete course %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	data, err := client.DeleteCourse(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	return reportWrite(g, data, "Deleted course %s", c.ID)
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
func formatCourseRows(sys units.System, courses []map[string]any) [][]string {
	rows := make([][]string, 0, len(courses))
	for _, c := range courses {
		rows = append(rows, []string{
			jsonString(c, "courseId"),
			jsonString(c, "courseName"),
			courseTypeKey(c),
			sys.Distance(jsonFloat(c, "distanceInMeters")),
			formatDate(jsonString(c, "createdDate")),
		})
	}
	return rows
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
