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

var activitiesEndpoints = []Endpoint{
	{
		Name: "activities.list", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/activitylist-service/activities/search/activities?start={start}&limit={limit}&activityType={activityType}",
		Params: []Param{{Name: "activityType", Optional: true}},
	},
	{Name: "activities.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}"},
	{Name: "activities.details", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/details?maxChartSize={maxChart}&maxPolylineSize={maxPoly}"},
	{Name: "activities.splits", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/splits"},
	{Name: "activities.typed-splits", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/typedsplits"},
	{Name: "activities.split-summaries", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/split_summaries"},
	{Name: "activities.weather", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/weather"},
	{Name: "activities.hr-zones", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/hrTimeInZones"},
	{Name: "activities.power-zones", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/powerTimeInZones"},
	{Name: "activities.exercise-sets", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/exerciseSets"},
	{Name: "activities.gear", Method: http.MethodGet, Host: ConnectAPI, Path: "/gear-service/gear?activityId={activityID}"},
	{
		Name: "activities.search", Method: http.MethodGet, Host: ConnectAPI,
		Path:   "/activitylist-service/activities/search/activities?start={start}&limit={limit}&startDate={startDate}&endDate={endDate}",
		Params: []Param{{Name: "endDate", Optional: true}},
	},
	{Name: "activities.set-exercise-sets", Method: http.MethodPut, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}/exerciseSets", Body: true},
	{Name: "activities.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/activity-service/activity", Body: true},
	{Name: "activities.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}", Body: true},
	{Name: "activities.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/activity-service/activity/{activityID}"},
	{Name: "activities.types", Method: http.MethodGet, Host: ConnectAPI, Path: "/activity-service/activity/activityTypes"},
	{Name: "activities.pending", Method: http.MethodGet, Host: ConnectAPI, Path: "/pendingactivity-service/pending-activities"},
}

// CountActivities returns the total number of activities for the authenticated user.
func (c *Client) CountActivities(ctx context.Context) (int, error) {
	data, err := c.ConnectAPI(ctx, http.MethodGet, "/activitylist-service/activities/count", nil)
	if err != nil {
		return 0, err
	}

	var result struct {
		TotalCount int `json:"totalCount"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("parse activity count: %w", err)
	}
	return result.TotalCount, nil
}

// downloadPath returns the API path for downloading an activity in the given format.
func downloadPath(activityID string, format DownloadFormat) (string, error) {
	switch format {
	case FormatFIT:
		return "/download-service/files/activity/" + activityID, nil
	case FormatTCX:
		return "/download-service/export/tcx/activity/" + activityID, nil
	case FormatGPX:
		return "/download-service/export/gpx/activity/" + activityID, nil
	case FormatKML:
		return "/download-service/export/kml/activity/" + activityID, nil
	case FormatCSV:
		return "/download-service/export/csv/activity/" + activityID, nil
	default:
		return "", &InvalidFileFormatError{Format: string(format)}
	}
}

// DownloadActivity downloads an activity in the specified format.
func (c *Client) DownloadActivity(ctx context.Context, activityID string, format DownloadFormat) ([]byte, error) {
	path, err := downloadPath(activityID, format)
	if err != nil {
		return nil, err
	}
	return c.Download(ctx, path)
}

// UploadActivity uploads an activity file (FIT, GPX, or TCX).
func (c *Client) UploadActivity(ctx context.Context, filePath string) (json.RawMessage, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	switch ext {
	case "fit", "gpx", "tcx":
		// supported
	default:
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

	return c.doUpload(ctx, "/upload-service/upload", writer.FormDataContentType(), &buf, true)
}

func (c *Client) doUpload(ctx context.Context, path, contentType string, body io.Reader, canRetry bool) (json.RawMessage, error) {
	r := request{kind: kindUpload, method: http.MethodPost, path: path, contentType: contentType, noRefresh: !canRetry}
	if body != nil {
		var err error
		r.body, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read upload body: %w", err)
		}
	}
	resp, err := c.do(ctx, r)
	if err != nil {
		return nil, err
	}
	if len(resp.body) == 0 {
		return nil, nil
	}
	return json.RawMessage(resp.body), nil
}

// CreateManualActivity creates a manual activity entry.
func (c *Client) CreateManualActivity(ctx context.Context, name, activityType, timezone string, distanceMeters float64, durationSeconds float64, startTime string) (json.RawMessage, error) {
	payload := map[string]any{
		"activityTypeDTO":      map[string]any{"typeKey": activityType},
		"accessControlRuleDTO": map[string]any{"typeId": 2, "typeKey": "private"},
		"timeZoneUnitDTO":      map[string]any{"unitKey": timezone},
		"activityName":         name,
		"metadataDTO":          map[string]any{"autoCalcCalories": true},
		"summaryDTO": map[string]any{
			"startTimeLocal": startTime,
			"distance":       distanceMeters,
			"duration":       durationSeconds,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal activity: %w", err)
	}

	return c.Call(ctx, "activities.create", nil, body)
}

// RenameActivity updates the name of an activity.
func (c *Client) RenameActivity(ctx context.Context, activityID string, name string) (json.RawMessage, error) {
	id, err := strconv.ParseInt(activityID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID: %w", err)
	}

	payload := map[string]any{
		"activityId":   id,
		"activityName": name,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal rename: %w", err)
	}

	return c.Call(ctx, "activities.update", map[string]string{"activityID": activityID}, body)
}

// RetypeActivity changes the activity type.
func (c *Client) RetypeActivity(ctx context.Context, activityID string, typeID int, typeKey string, parentTypeID int) (json.RawMessage, error) {
	id, err := strconv.ParseInt(activityID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID: %w", err)
	}

	payload := map[string]any{
		"activityId": id,
		"activityTypeDTO": map[string]any{
			"typeId":       typeID,
			"typeKey":      typeKey,
			"parentTypeId": parentTypeID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal retype: %w", err)
	}

	return c.Call(ctx, "activities.update", map[string]string{"activityID": activityID}, body)
}
