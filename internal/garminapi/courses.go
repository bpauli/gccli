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

// downloadPath returns the API path for downloading a course in the given format.
func downloadCoursePath(courseID string, format ActivityDownloadFormat) (string, error) {
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
func (c *Client) DownloadCourse(ctx context.Context, courseID string, format ActivityDownloadFormat) ([]byte, error) {
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

// DeleteCourse deletes a course by ID.
func (c *Client) DeleteCourse(ctx context.Context, courseID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/course-service/course/"+courseID, nil)
	return err
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
