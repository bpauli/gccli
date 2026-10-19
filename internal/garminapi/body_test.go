package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBodyEndpoints(t *testing.T) {
	rng := map[string]string{"startDate": "2024-01-01", "endDate": "2024-01-31"}
	entry := map[string]string{"date": "2024-01-15", "version": "12345"}
	testEndpoints(t, bodyEndpoints, []endpointCase{
		{"body.composition", rng, "GET /weight-service/weight/dateRange?startDate=2024-01-01&endDate=2024-01-31"},
		{"body.weigh-ins", rng, "GET /weight-service/weight/range/2024-01-01/2024-01-31?includeAll=true"},
		{"body.daily-weigh-ins", map[string]string{"date": "2024-01-15"}, "GET /weight-service/weight/dayview/2024-01-15?includeAll=true"},
		{"body.latest-weight", map[string]string{"date": "2024-01-15"}, "GET /weight-service/weight/latest?date=2024-01-15"},
		{"body.delete-weight", entry, "DELETE /weight-service/weight/2024-01-15/byversion/12345"},
		{"body.blood-pressure", rng, "GET /bloodpressure-service/bloodpressure/range/2024-01-01/2024-01-31?includeAll=true"},
		{"body.delete-blood-pressure", entry, "DELETE /bloodpressure-service/bloodpressure/2024-01-15/12345"},
		{"body.add-weight", nil, "POST /weight-service/user-weight"},
		{"body.add-blood-pressure", nil, "POST /bloodpressure-service/bloodpressure"},
	})
}

// --- AddWeight tests ---

func TestAddWeight_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/weight-service/user-weight" {
			t.Errorf("path = %s, want /weight-service/user-weight", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload["value"] != 75.5 {
			t.Errorf("value = %v, want 75.5", payload["value"])
		}
		if payload["unitKey"] != "kg" {
			t.Errorf("unitKey = %v, want kg", payload["unitKey"])
		}
		if payload["sourceType"] != "MANUAL" {
			t.Errorf("sourceType = %v, want MANUAL", payload["sourceType"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"weight":75.5}`))
	})

	_, client := testServer(t, handler)
	data, err := client.AddWeight(context.Background(), 75.5, "kg", "2024-01-15T10:00:00", "2024-01-15T09:00:00")
	if err != nil {
		t.Fatalf("AddWeight: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["weight"] != 75.5 {
		t.Errorf("weight = %v, want 75.5", result["weight"])
	}
}

func TestAddWeight_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.AddWeight(context.Background(), 75.5, "kg", "2024-01-15T10:00:00", "2024-01-15T09:00:00")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- AddBloodPressure tests ---

func TestAddBloodPressure_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/bloodpressure-service/bloodpressure" {
			t.Errorf("path = %s, want /bloodpressure-service/bloodpressure", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload["systolic"] != float64(120) {
			t.Errorf("systolic = %v, want 120", payload["systolic"])
		}
		if payload["diastolic"] != float64(80) {
			t.Errorf("diastolic = %v, want 80", payload["diastolic"])
		}
		if payload["pulse"] != float64(72) {
			t.Errorf("pulse = %v, want 72", payload["pulse"])
		}
		if payload["sourceType"] != "MANUAL" {
			t.Errorf("sourceType = %v, want MANUAL", payload["sourceType"])
		}
		if payload["notes"] != "morning reading" {
			t.Errorf("notes = %v, want 'morning reading'", payload["notes"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"systolic":120,"diastolic":80,"pulse":72}`))
	})

	_, client := testServer(t, handler)
	data, err := client.AddBloodPressure(context.Background(), 120, 80, 72, "2024-01-15T10:00:00", "2024-01-15T09:00:00", "morning reading")
	if err != nil {
		t.Fatalf("AddBloodPressure: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["systolic"] != float64(120) {
		t.Errorf("systolic = %v, want 120", result["systolic"])
	}
}

func TestAddBloodPressure_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.AddBloodPressure(context.Background(), 120, 80, 72, "ts", "gmt", "")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- UploadBodyComposition tests ---

func TestUploadBodyComposition_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/upload-service/upload" {
			t.Errorf("path = %s, want /upload-service/upload", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %s, want multipart/form-data", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("get form file: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "body_composition.fit" {
			t.Errorf("filename = %s, want body_composition.fit", header.Filename)
		}

		data, _ := io.ReadAll(file)
		if len(data) == 0 {
			t.Error("expected non-empty FIT data")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detailedImportResult":{"successes":[]}}`))
	})

	_, client := testServer(t, handler)
	fitData := []byte("fake-fit-data-for-test")
	data, err := client.UploadBodyComposition(context.Background(), fitData)
	if err != nil {
		t.Fatalf("UploadBodyComposition: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["detailedImportResult"] == nil {
		t.Error("expected detailedImportResult field")
	}
}

func TestUploadBodyComposition_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.UploadBodyComposition(context.Background(), []byte("fit-data"))
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
