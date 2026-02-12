package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
)

// GetBodyComposition returns body composition data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetBodyComposition(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/weight-service/weight/dateRange?startDate=%s&endDate=%s", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWeighIns returns weigh-in data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetWeighIns(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/weight-service/weight/range/%s/%s?includeAll=true", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetDailyWeighIns returns weigh-in data for a single day.
// Date format: YYYY-MM-DD.
func (c *Client) GetDailyWeighIns(ctx context.Context, date string) (json.RawMessage, error) {
	path := fmt.Sprintf("/weight-service/weight/dayview/%s?includeAll=true", date)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// AddWeight adds a weigh-in entry.
func (c *Client) AddWeight(ctx context.Context, weight float64, unitKey string, dateTimestamp string, gmtTimestamp string) (json.RawMessage, error) {
	payload := map[string]any{
		"dateTimestamp": dateTimestamp,
		"gmtTimestamp":  gmtTimestamp,
		"unitKey":       unitKey,
		"value":         weight,
		"sourceType":    "MANUAL",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal weight: %w", err)
	}

	return c.ConnectAPI(ctx, http.MethodPost, "/weight-service/user-weight", bytes.NewReader(body))
}

// DeleteWeight deletes a weigh-in entry by date and version (primary key).
// Date format: YYYY-MM-DD.
func (c *Client) DeleteWeight(ctx context.Context, date string, version string) error {
	path := fmt.Sprintf("/weight-service/weight/%s/byversion/%s", date, version)
	_, err := c.ConnectAPI(ctx, http.MethodDelete, path, nil)
	return err
}

// GetBloodPressure returns blood pressure data for a date range.
// Date format: YYYY-MM-DD.
func (c *Client) GetBloodPressure(ctx context.Context, startDate, endDate string) (json.RawMessage, error) {
	path := fmt.Sprintf("/bloodpressure-service/bloodpressure/range/%s/%s?includeAll=true", startDate, endDate)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// AddBloodPressure adds a blood pressure measurement.
func (c *Client) AddBloodPressure(ctx context.Context, systolic, diastolic, pulse int, timestampLocal, timestampGMT, notes string) (json.RawMessage, error) {
	payload := map[string]any{
		"measurementTimestampLocal": timestampLocal,
		"measurementTimestampGMT":   timestampGMT,
		"systolic":                  systolic,
		"diastolic":                 diastolic,
		"pulse":                     pulse,
		"sourceType":                "MANUAL",
		"notes":                     notes,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal blood pressure: %w", err)
	}

	return c.ConnectAPI(ctx, http.MethodPost, "/bloodpressure-service/bloodpressure", bytes.NewReader(body))
}

// DeleteBloodPressure deletes a blood pressure entry by date and version.
// Date format: YYYY-MM-DD.
func (c *Client) DeleteBloodPressure(ctx context.Context, date string, version string) error {
	path := fmt.Sprintf("/bloodpressure-service/bloodpressure/%s/%s", date, version)
	_, err := c.ConnectAPI(ctx, http.MethodDelete, path, nil)
	return err
}

// UploadBodyComposition uploads a FIT-encoded body composition file.
func (c *Client) UploadBodyComposition(ctx context.Context, fitData []byte) (json.RawMessage, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "body_composition.fit")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fitData); err != nil {
		return nil, fmt.Errorf("write FIT data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	return c.doUpload(ctx, "/upload-service/upload", writer.FormDataContentType(), &buf, true)
}
