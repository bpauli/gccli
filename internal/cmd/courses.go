package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

var courseColumns = []Column{
	{Label: "ID", Path: "courseId"},
	{Label: "NAME", Path: "courseName"},
	{Label: "TYPE", Path: "courseType.typeKey"},
	{Label: "DISTANCE", Path: "distanceInMeters", Format: Distance},
	{Label: "CREATED", Path: "createdDate", Format: DateOnly},
}

var coursesGroup = Group{Name: "courses", Help: "Manage courses.", Children: []Node{
	Leaf{
		Name: "list", Help: "List courses.", Default: true,
		Endpoint: "courses.list", Verb: "list courses",
		Output: Rows{Noun: "courses", Items: "coursesForUser", Columns: courseColumns},
	},
	Leaf{
		Name: "favorites", Help: "List favorite courses.",
		Endpoint: "courses.favorites", Verb: "list favorite courses",
		Output: Rows{Noun: "favorite courses", Columns: courseColumns},
	},
	Custom{Name: "detail", Help: "View course details.", Cmd: &CourseDetailCmd{}},
	Custom{Name: "download", Help: "Download course file.", Cmd: &CourseDownloadCmd{}},
	Custom{Name: "import", Help: "Import a GPX file as a new course.", Cmd: &CourseImportCmd{}},
	Custom{Name: "send", Help: "Send a course to a device.", Cmd: &CourseSendCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a course.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Course ID."`
		}{}),
		Endpoint: "courses.delete",
		Params:   Params{"courseID": In("ID")},
		Verb:     "delete course",
		Output:   Write{Done: "Deleted course {ID}", Confirm: "Delete course {ID}?"},
	},
}}

// CourseDetailCmd shows details for a course.
type CourseDetailCmd struct {
	ID ID `arg:"" help:"Course ID."`
}

func (c *CourseDetailCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "courses.get", map[string]string{"courseID": string(c.ID)}, nil)
	if err != nil {
		return fmt.Errorf("get course: %w", err)
	}

	return writeObject(g, data)
}

// CourseDownloadCmd downloads a course in a specified format.
type CourseDownloadCmd struct {
	ID     ID     `arg:"" help:"Course ID."`
	Format string `help:"Download format: fit, gpx." default:"fit" enum:"fit,gpx"`
	Output string `help:"Output file path (default: course_{id}.{format})." short:"o"`
}

func (c *CourseDownloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	format := garminapi.DownloadFormat(strings.ToLower(c.Format))

	data, err := client.DownloadCourse(g.Context, string(c.ID), format)
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
	Type    string `help:"Activity type key: cycling, e_bike_fitness, e_bike_mountain, gravel_cycling, hiking, mountain_biking, other, road_biking, running, street_running, trail_running, ultra_run or walking." short:"t" default:"cycling" enum:"cycling,e_bike_fitness,e_bike_mountain,gravel_cycling,hiking,mountain_biking,other,road_biking,running,street_running,trail_running,ultra_run,walking"`
	Privacy int    `help:"Course privacy: 1=public, 2=private, 4=group." short:"p" default:"2" enum:"1,2,4"`
}

func (c *CourseImportCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	g.UI.Infof("Importing %s...", filepath.Base(c.File))
	saved, err := client.ImportCourse(g.Context, c.File, garminapi.CourseImportOptions{
		Name:           c.Name,
		ActivityTypePK: courseActivityTypes[c.Type],
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
	CourseID NumericID `arg:"" help:"Course ID."`
	DeviceID NumericID `arg:"" help:"Device ID."`
}

func (c *CourseSendCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	// Fetch course detail to get the course name.
	courseData, err := client.Call(g.Context, "courses.get", map[string]string{"courseID": c.CourseID.String()}, nil)
	if err != nil {
		return fmt.Errorf("get course: %w", err)
	}

	var course map[string]any
	if err := json.Unmarshal(courseData, &course); err != nil {
		return fmt.Errorf("parse course: %w", err)
	}
	courseName := jsonString(course, "courseName")

	data, err := client.SendCourseToDevice(g.Context, int64(c.CourseID), int64(c.DeviceID), courseName)
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
