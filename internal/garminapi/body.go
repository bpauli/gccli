package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
)

var bodyEndpoints = []Endpoint{
	{Name: "body.composition", Method: http.MethodGet, Host: ConnectAPI, Path: "/weight-service/weight/dateRange?startDate={startDate}&endDate={endDate}"},
	{Name: "body.weigh-ins", Method: http.MethodGet, Host: ConnectAPI, Path: "/weight-service/weight/range/{startDate}/{endDate}?includeAll=true"},
	{Name: "body.daily-weigh-ins", Method: http.MethodGet, Host: ConnectAPI, Path: "/weight-service/weight/dayview/{date}?includeAll=true"},
	{Name: "body.latest-weight", Method: http.MethodGet, Host: ConnectAPI, Path: "/weight-service/weight/latest?date={date}"},
	{Name: "body.delete-weight", Method: http.MethodDelete, Host: ConnectAPI, Path: "/weight-service/weight/{date}/byversion/{version}"},
	{Name: "body.blood-pressure", Method: http.MethodGet, Host: ConnectAPI, Path: "/bloodpressure-service/bloodpressure/range/{startDate}/{endDate}?includeAll=true"},
	{Name: "body.glucose", Method: http.MethodGet, Host: ConnectAPI, Path: "/biometric-service/biometric/glucose/{date}"},
	{Name: "body.delete-blood-pressure", Method: http.MethodDelete, Host: ConnectAPI, Path: "/bloodpressure-service/bloodpressure/{date}/{version}"},
	{Name: "body.add-weight", Method: http.MethodPost, Host: ConnectAPI, Path: "/weight-service/user-weight", Body: true},
	{Name: "body.add-blood-pressure", Method: http.MethodPost, Host: ConnectAPI, Path: "/bloodpressure-service/bloodpressure", Body: true},
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

	return c.Call(ctx, "body.add-weight", nil, body)
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

	return c.Call(ctx, "body.add-blood-pressure", nil, body)
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
