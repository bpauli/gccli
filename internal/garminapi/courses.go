package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetCourses returns all courses owned by the current user.
func (c *Client) GetCourses(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/web-gateway/course/owner/", nil)
}

// GetCourse returns details for a specific course.
func (c *Client) GetCourse(ctx context.Context, courseID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/course-service/course/"+courseID, nil)
}

// GetCourseFavorites returns the user's favorite courses.
func (c *Client) GetCourseFavorites(ctx context.Context) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/course-service/course/favorites", nil)
}

// downloadCoursePath returns the API path for downloading a course in the given format.
func downloadCoursePath(courseID string, format DownloadFormat) (string, error) {
	switch format {
	case FormatFIT:
		return "/course-service/course/fit/" + courseID + "/0?elevation=true", nil
	case FormatGPX:
		return "/course-service/course/gpx/" + courseID, nil
	default:
		return "", &InvalidFileFormatError{Format: string(format)}
	}
}

// DownloadCourse downloads a course in the specified format.
func (c *Client) DownloadCourse(ctx context.Context, courseID string, format DownloadFormat) ([]byte, error) {
	path, err := downloadCoursePath(courseID, format)
	if err != nil {
		return nil, err
	}
	return c.Download(ctx, path)
}

// ImportCourseGPX uploads a GPX file and returns parsed course data.
func (c *Client) ImportCourseGPX(ctx context.Context, filePath string) (json.RawMessage, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext != "gpx" {
		return nil, &InvalidFileFormatError{Format: ext}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	return c.doUpload(ctx, "/course-service/course/import", writer.FormDataContentType(), &buf, true)
}

// GetCourseElevation enriches geo points with elevation data.
func (c *Client) GetCourseElevation(ctx context.Context, points json.RawMessage) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/course-service/course/elevation?smoothingEnabled=true", bytes.NewReader(points))
}

// SaveCourse creates a new course from enriched course data.
// The course payload must include coursePrivacy (1=public, 2=private, 4=group)
// and rulePK (must not be null, typically 2).
func (c *Client) SaveCourse(ctx context.Context, course json.RawMessage) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/course-service/course", bytes.NewReader(course))
}

// CourseImportOptions are the course fields that ImportCourse sets on the
// imported course before it saves it.
type CourseImportOptions struct {
	// Name replaces the course name from the GPX file. When Name and the
	// GPX name are both empty, the file name without its extension is used.
	Name           string
	ActivityTypePK int
	// Privacy is 1 (public), 2 (private) or 4 (group).
	Privacy int
}

// ImportCourse creates a course from a GPX file. It imports the file,
// adds the elevation of each geo point, and saves the course. It returns
// the save response.
func (c *Client) ImportCourse(ctx context.Context, path string, opts CourseImportOptions) (json.RawMessage, error) {
	parsed, err := c.ImportCourseGPX(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("import GPX: %w", err)
	}

	var course map[string]any
	if err := json.Unmarshal(parsed, &course); err != nil {
		return nil, fmt.Errorf("parse imported course: %w", err)
	}

	geoPoints, ok := course["geoPoints"].([]any)
	if !ok || len(geoPoints) == 0 {
		return nil, fmt.Errorf("imported course has no geo points")
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
		return nil, fmt.Errorf("marshal elevation input: %w", err)
	}

	elevData, err := c.GetCourseElevation(ctx, elevJSON)
	if err != nil {
		return nil, fmt.Errorf("get elevation: %w", err)
	}

	var elevPoints [][]any
	if err := json.Unmarshal(elevData, &elevPoints); err != nil {
		return nil, fmt.Errorf("parse elevation: %w", err)
	}

	for i, ep := range elevPoints {
		if i >= len(geoPoints) {
			break
		}
		if pt, ok := geoPoints[i].(map[string]any); ok && len(ep) >= 3 {
			pt["elevation"] = ep[2]
		}
	}

	if opts.Name != "" {
		course["courseName"] = opts.Name
	} else if name := course["courseName"]; name == nil || name == "" {
		course["courseName"] = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	course["activityTypePk"] = opts.ActivityTypePK
	if course["coordinateSystem"] == nil {
		course["coordinateSystem"] = "WGS84"
	}
	if course["sourceTypeId"] == nil {
		course["sourceTypeId"] = 3
	}
	if course["startPoint"] == nil {
		course["startPoint"] = geoPoints[0]
	}

	save := filterCourseFields(course)
	save["coursePrivacy"] = opts.Privacy
	if save["rulePK"] == nil {
		save["rulePK"] = 2
	}

	payload, err := json.Marshal(save)
	if err != nil {
		return nil, fmt.Errorf("marshal course: %w", err)
	}

	saved, err := c.SaveCourse(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("save course: %w", err)
	}
	return saved, nil
}

var courseSaveFields = []string{
	"activityTypePk", "boundingBox", "coordinateSystem", "courseLines",
	"courseName", "coursePoints", "distanceMeter", "elapsedSeconds",
	"elevationGainMeter", "elevationLossMeter", "favorite", "geoPoints",
	"hasPaceBand", "hasPowerGuide", "hasTurnDetectionDisabled", "includeLaps",
	"matchedToSegments", "openStreetMap", "rulePK", "sourceTypeId",
	"speedMeterPerSecond", "startPoint", "userProfilePk",
}

func filterCourseFields(course map[string]any) map[string]any {
	save := make(map[string]any, len(courseSaveFields))
	for _, key := range courseSaveFields {
		if v, ok := course[key]; ok && v != nil {
			save[key] = v
		}
	}
	return save
}

// DeleteCourse deletes a course by ID.
func (c *Client) DeleteCourse(ctx context.Context, courseID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodDelete, "/course-service/course/"+courseID, nil)
}

// SendCourseToDevice sends a course to a device via the device message API.
func (c *Client) SendCourseToDevice(ctx context.Context, courseID, deviceID, courseName string) (json.RawMessage, error) {
	deviceIDInt, err := strconv.ParseInt(deviceID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse device ID: %w", err)
	}

	courseIDInt, err := strconv.ParseInt(courseID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse course ID: %w", err)
	}

	payload := []map[string]any{
		{
			"deviceId":    deviceIDInt,
			"messageUrl":  fmt.Sprintf("course-service/course/fit/%s/%s?elevation=true", courseID, deviceID),
			"messageType": "courses",
			"messageName": courseName,
			"groupName":   nil,
			"priority":    0,
			"fileType":    "FIT",
			"metaDataId":  courseIDInt,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal send payload: %w", err)
	}

	return c.ConnectAPI(ctx, http.MethodPost, "/device-service/devicemessage/messages", bytes.NewReader(body))
}
