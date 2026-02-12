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

// --- GetBodyComposition tests ---

func TestGetBodyComposition_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weight-service/weight/dateRange" {
			t.Errorf("path = %s, want /weight-service/weight/dateRange", r.URL.Path)
		}
		if got := r.URL.Query().Get("startDate"); got != "2024-01-01" {
			t.Errorf("startDate = %s, want 2024-01-01", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-01-31" {
			t.Errorf("endDate = %s, want 2024-01-31", got)
		}
		_, _ = w.Write([]byte(`{"totalAverage":{"weight":75000.0}}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBodyComposition(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetBodyComposition: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["totalAverage"] == nil {
		t.Error("expected totalAverage field")
	}
}

func TestGetBodyComposition_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetBodyComposition(context.Background(), "2024-01-01", "2024-01-31")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetWeighIns tests ---

func TestGetWeighIns_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weight-service/weight/range/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /weight-service/weight/range/2024-01-01/2024-01-31", r.URL.Path)
		}
		if got := r.URL.Query().Get("includeAll"); got != "true" {
			t.Errorf("includeAll = %s, want true", got)
		}
		_, _ = w.Write([]byte(`{"dailyWeightSummaries":[{"date":"2024-01-15"}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWeighIns(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetWeighIns: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["dailyWeightSummaries"] == nil {
		t.Error("expected dailyWeightSummaries field")
	}
}

// --- GetDailyWeighIns tests ---

func TestGetDailyWeighIns_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weight-service/weight/dayview/2024-01-15" {
			t.Errorf("path = %s, want /weight-service/weight/dayview/2024-01-15", r.URL.Path)
		}
		if got := r.URL.Query().Get("includeAll"); got != "true" {
			t.Errorf("includeAll = %s, want true", got)
		}
		_, _ = w.Write([]byte(`{"dateWeightList":[{"weight":75.0}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDailyWeighIns(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetDailyWeighIns: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["dateWeightList"] == nil {
		t.Error("expected dateWeightList field")
	}
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

// --- DeleteWeight tests ---

func TestDeleteWeight_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/weight-service/weight/2024-01-15/byversion/12345" {
			t.Errorf("path = %s, want /weight-service/weight/2024-01-15/byversion/12345", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	err := client.DeleteWeight(context.Background(), "2024-01-15", "12345")
	if err != nil {
		t.Fatalf("DeleteWeight: %v", err)
	}
}

func TestDeleteWeight_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, client := testServer(t, handler)
	err := client.DeleteWeight(context.Background(), "2024-01-15", "12345")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetBloodPressure tests ---

func TestGetBloodPressure_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bloodpressure-service/bloodpressure/range/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /bloodpressure-service/bloodpressure/range/2024-01-01/2024-01-31", r.URL.Path)
		}
		if got := r.URL.Query().Get("includeAll"); got != "true" {
			t.Errorf("includeAll = %s, want true", got)
		}
		_, _ = w.Write([]byte(`{"measurementSummaries":[{"systolic":120,"diastolic":80}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBloodPressure(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetBloodPressure: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["measurementSummaries"] == nil {
		t.Error("expected measurementSummaries field")
	}
}

func TestGetBloodPressure_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetBloodPressure(context.Background(), "2024-01-01", "2024-01-31")
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

// --- DeleteBloodPressure tests ---

func TestDeleteBloodPressure_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/bloodpressure-service/bloodpressure/2024-01-15/67890" {
			t.Errorf("path = %s, want /bloodpressure-service/bloodpressure/2024-01-15/67890", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	err := client.DeleteBloodPressure(context.Background(), "2024-01-15", "67890")
	if err != nil {
		t.Fatalf("DeleteBloodPressure: %v", err)
	}
}

func TestDeleteBloodPressure_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, client := testServer(t, handler)
	err := client.DeleteBloodPressure(context.Background(), "2024-01-15", "67890")
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
