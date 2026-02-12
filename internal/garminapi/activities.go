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

// ActivityDownloadFormat represents a supported activity export format.
type ActivityDownloadFormat string

const (
	FormatFIT ActivityDownloadFormat = "fit"
	FormatTCX ActivityDownloadFormat = "tcx"
	FormatGPX ActivityDownloadFormat = "gpx"
	FormatKML ActivityDownloadFormat = "kml"
	FormatCSV ActivityDownloadFormat = "csv"
)

// CountActivities returns the total number of activities for the authenticated user.
func (c *Client) CountActivities(ctx context.Context) (int, error) {
	data, err := c.ConnectAPI(ctx, http.MethodGet, "/activitylist-service/activities/count", nil)
	if err != nil {
		return 0, err
	}

	var count int
	if err := json.Unmarshal(data, &count); err != nil {
		return 0, fmt.Errorf("parse activity count: %w", err)
	}
	return count, nil
}

// GetActivities returns a page of activities.
func (c *Client) GetActivities(ctx context.Context, start, limit int, activityType string) (json.RawMessage, error) {
	path := fmt.Sprintf("/activitylist-service/activities/search/activities?start=%d&limit=%d", start, limit)
	if activityType != "" {
		path += "&activityType=" + activityType
	}
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetActivity returns a single activity by ID.
func (c *Client) GetActivity(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID, nil)
}

// GetActivityDetails returns detailed activity data including charts and polylines.
func (c *Client) GetActivityDetails(ctx context.Context, activityID string, maxChart, maxPoly int) (json.RawMessage, error) {
	path := fmt.Sprintf("/activity-service/activity/%s/details?maxChartSize=%d&maxPolylineSize=%d",
		activityID, maxChart, maxPoly)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetActivitySplits returns the splits for an activity.
func (c *Client) GetActivitySplits(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/splits", nil)
}

// GetActivityTypedSplits returns the typed splits for an activity.
func (c *Client) GetActivityTypedSplits(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/typedsplits", nil)
}

// GetActivitySplitSummaries returns the split summaries for an activity.
func (c *Client) GetActivitySplitSummaries(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/split_summaries", nil)
}

// GetActivityWeather returns weather data for an activity.
func (c *Client) GetActivityWeather(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/weather", nil)
}

// GetActivityHRZones returns heart rate time-in-zone data for an activity.
func (c *Client) GetActivityHRZones(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/hrTimeInZones", nil)
}

// GetActivityPowerZones returns power time-in-zone data for an activity.
func (c *Client) GetActivityPowerZones(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/powerTimeInZones", nil)
}

// GetActivityExerciseSets returns exercise set data for strength training activities.
func (c *Client) GetActivityExerciseSets(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/activity-service/activity/"+activityID+"/exerciseSets", nil)
}

// GetActivityGear returns gear linked to an activity.
func (c *Client) GetActivityGear(ctx context.Context, activityID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/gear-service/gear?activityId="+activityID, nil)
}

// SearchActivities returns activities matching date range and optional filters.
func (c *Client) SearchActivities(ctx context.Context, start, limit int, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/activitylist-service/activities/search/activities?start=%d&limit=%d&startDate=%s",
		start, limit, startDate)
	if endDate != "" {
		path += "&endDate=" + endDate
	}
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// downloadPath returns the API path for downloading an activity in the given format.
func downloadPath(activityID string, format ActivityDownloadFormat) (string, error) {
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
func (c *Client) DownloadActivity(ctx context.Context, activityID string, format ActivityDownloadFormat) ([]byte, error) {
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
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read upload body: %w", err)
		}
	}
	return c.doUploadBytes(ctx, path, contentType, bodyBytes, canRetry)
}

func (c *Client) doUploadBytes(ctx context.Context, path, contentType string, body []byte, canRetry bool) (json.RawMessage, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.tokens.OAuth2AccessToken)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "com.garmin.android.apps.connectmobile")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, resp.Body)
		if canRetry {
			if refreshErr := c.refreshToken(ctx); refreshErr == nil {
				return c.doUploadBytes(ctx, path, contentType, body, false)
			}
		}
		return nil, &TokenExpiredError{Email: c.tokens.Email}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &RateLimitError{RetryAfter: resp.Header.Get("Retry-After")}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upload response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &GarminAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(data),
		}
	}

	if len(data) == 0 {
		return nil, nil
	}

	return json.RawMessage(data), nil
}

// CreateManualActivity creates a manual activity entry.
func (c *Client) CreateManualActivity(ctx context.Context, name, activityType string, distanceMeters float64, durationSeconds float64, startTime string) (json.RawMessage, error) {
	payload := map[string]any{
		"activityTypeDTO":      map[string]any{"typeKey": activityType},
		"accessControlRuleDTO": map[string]any{"typeId": 2, "typeKey": "private"},
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

	return c.ConnectAPI(ctx, http.MethodPost, "/activity-service/activity", bytes.NewReader(body))
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

	return c.ConnectAPI(ctx, http.MethodPut, "/activity-service/activity/"+activityID, bytes.NewReader(body))
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

	return c.ConnectAPI(ctx, http.MethodPut, "/activity-service/activity/"+activityID, bytes.NewReader(body))
}

// DeleteActivity deletes an activity by ID.
func (c *Client) DeleteActivity(ctx context.Context, activityID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/activity-service/activity/"+activityID, nil)
	return err
}
