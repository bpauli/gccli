package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// --- GetDailySummary tests ---

func TestGetDailySummary_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/usersummary/daily/testuser" {
			t.Errorf("path = %s, want /usersummary-service/usersummary/daily/testuser", r.URL.Path)
		}
		if got := r.URL.Query().Get("calendarDate"); got != "2024-01-15" {
			t.Errorf("calendarDate = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`{"totalSteps":10000,"totalDistanceMeters":8000}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDailySummary(context.Background(), "testuser", "2024-01-15")
	if err != nil {
		t.Fatalf("GetDailySummary: %v", err)
	}

	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if summary["totalSteps"] != float64(10000) {
		t.Errorf("totalSteps = %v, want 10000", summary["totalSteps"])
	}
}

func TestGetDailySummary_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetDailySummary(context.Background(), "testuser", "2024-01-15")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}

// --- GetSteps tests ---

func TestGetSteps_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailySummaryChart/testuser" {
			t.Errorf("path = %s, want /wellness-service/wellness/dailySummaryChart/testuser", r.URL.Path)
		}
		if got := r.URL.Query().Get("date"); got != "2024-01-15" {
			t.Errorf("date = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`{"totalSteps":8500}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetSteps(context.Background(), "testuser", "2024-01-15")
	if err != nil {
		t.Fatalf("GetSteps: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["totalSteps"] != float64(8500) {
		t.Errorf("totalSteps = %v, want 8500", result["totalSteps"])
	}
}

// --- GetDailySteps tests ---

func TestGetDailySteps_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/stats/steps/daily/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /usersummary-service/stats/steps/daily/2024-01-01/2024-01-31", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"calendarDate":"2024-01-01","totalSteps":9000}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetDailySteps(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetDailySteps: %v", err)
	}

	var steps []map[string]any
	if err := json.Unmarshal(data, &steps); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("got %d entries, want 1", len(steps))
	}
}

// --- GetWeeklySteps tests ---

func TestGetWeeklySteps_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/stats/steps/weekly/2024-01-31/4" {
			t.Errorf("path = %s, want /usersummary-service/stats/steps/weekly/2024-01-31/4", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"week":"2024-W04","totalSteps":63000}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWeeklySteps(context.Background(), "2024-01-31", 4)
	if err != nil {
		t.Fatalf("GetWeeklySteps: %v", err)
	}

	var weeks []map[string]any
	if err := json.Unmarshal(data, &weeks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(weeks) != 1 {
		t.Errorf("got %d entries, want 1", len(weeks))
	}
}

// --- GetHeartRate tests ---

func TestGetHeartRate_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailyHeartRate/testuser" {
			t.Errorf("path = %s, want /wellness-service/wellness/dailyHeartRate/testuser", r.URL.Path)
		}
		if got := r.URL.Query().Get("date"); got != "2024-01-15" {
			t.Errorf("date = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`{"restingHeartRate":58,"maxHeartRate":165}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetHeartRate(context.Background(), "testuser", "2024-01-15")
	if err != nil {
		t.Fatalf("GetHeartRate: %v", err)
	}

	var hr map[string]any
	if err := json.Unmarshal(data, &hr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if hr["restingHeartRate"] != float64(58) {
		t.Errorf("restingHeartRate = %v, want 58", hr["restingHeartRate"])
	}
}

// --- GetRestingHeartRate tests ---

func TestGetRestingHeartRate_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/userstats-service/wellness/daily/testuser" {
			t.Errorf("path = %s, want /userstats-service/wellness/daily/testuser", r.URL.Path)
		}
		if got := r.URL.Query().Get("fromDate"); got != "2024-01-15" {
			t.Errorf("fromDate = %s, want 2024-01-15", got)
		}
		if got := r.URL.Query().Get("untilDate"); got != "2024-01-15" {
			t.Errorf("untilDate = %s, want 2024-01-15", got)
		}
		if got := r.URL.Query().Get("metricId"); got != "60" {
			t.Errorf("metricId = %s, want 60", got)
		}
		_, _ = w.Write([]byte(`{"allMetrics":[{"metricsMap":{"WELLNESS_RESTING_HEART_RATE":[58]}}]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRestingHeartRate(context.Background(), "testuser", "2024-01-15")
	if err != nil {
		t.Fatalf("GetRestingHeartRate: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetFloors tests ---

func TestGetFloors_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/floorsChartData/daily/2024-01-15" {
			t.Errorf("path = %s, want /wellness-service/wellness/floorsChartData/daily/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"floorsTSValue":{"floorsAscended":12}}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetFloors(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetFloors: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetSleep tests ---

func TestGetSleep_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailySleepData/testuser" {
			t.Errorf("path = %s, want /wellness-service/wellness/dailySleepData/testuser", r.URL.Path)
		}
		if got := r.URL.Query().Get("date"); got != "2024-01-15" {
			t.Errorf("date = %s, want 2024-01-15", got)
		}
		if got := r.URL.Query().Get("nonSleepBufferMinutes"); got != "60" {
			t.Errorf("nonSleepBufferMinutes = %s, want 60", got)
		}
		_, _ = w.Write([]byte(`{"sleepTimeSeconds":28800}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetSleep(context.Background(), "testuser", "2024-01-15")
	if err != nil {
		t.Fatalf("GetSleep: %v", err)
	}

	var sleep map[string]any
	if err := json.Unmarshal(data, &sleep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sleep["sleepTimeSeconds"] != float64(28800) {
		t.Errorf("sleepTimeSeconds = %v, want 28800", sleep["sleepTimeSeconds"])
	}
}

// --- GetStress tests ---

func TestGetStress_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailyStress/2024-01-15" {
			t.Errorf("path = %s, want /wellness-service/wellness/dailyStress/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"overallStressLevel":35}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetStress(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetStress: %v", err)
	}

	var stress map[string]any
	if err := json.Unmarshal(data, &stress); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stress["overallStressLevel"] != float64(35) {
		t.Errorf("overallStressLevel = %v, want 35", stress["overallStressLevel"])
	}
}

// --- GetWeeklyStress tests ---

func TestGetWeeklyStress_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/stats/stress/weekly/2024-01-31/4" {
			t.Errorf("path = %s, want /usersummary-service/stats/stress/weekly/2024-01-31/4", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"week":"2024-W04","averageStress":30}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWeeklyStress(context.Background(), "2024-01-31", 4)
	if err != nil {
		t.Fatalf("GetWeeklyStress: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetRespiration tests ---

func TestGetRespiration_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/respiration/2024-01-15" {
			t.Errorf("path = %s, want /wellness-service/wellness/daily/respiration/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"avgWakingRespirationValue":16.5}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRespiration(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetRespiration: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetSPO2 tests ---

func TestGetSPO2_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/spo2/2024-01-15" {
			t.Errorf("path = %s, want /wellness-service/wellness/daily/spo2/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"averageSPO2":97}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetSPO2(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetSPO2: %v", err)
	}

	var spo2 map[string]any
	if err := json.Unmarshal(data, &spo2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spo2["averageSPO2"] != float64(97) {
		t.Errorf("averageSPO2 = %v, want 97", spo2["averageSPO2"])
	}
}

// --- GetHRV tests ---

func TestGetHRV_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hrv-service/hrv/2024-01-15" {
			t.Errorf("path = %s, want /hrv-service/hrv/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"hrvSummary":{"weeklyAvg":45}}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetHRV(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetHRV: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetBodyBattery tests ---

func TestGetBodyBattery_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/bodyBattery/reports/daily" {
			t.Errorf("path = %s, want /wellness-service/wellness/bodyBattery/reports/daily", r.URL.Path)
		}
		if got := r.URL.Query().Get("startDate"); got != "2024-01-15" {
			t.Errorf("startDate = %s, want 2024-01-15", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-01-15" {
			t.Errorf("endDate = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-15","charged":65,"drained":40}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBodyBattery(context.Background(), "2024-01-15", "2024-01-15")
	if err != nil {
		t.Fatalf("GetBodyBattery: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

func TestGetBodyBattery_DateRange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("startDate"); got != "2024-01-01" {
			t.Errorf("startDate = %s, want 2024-01-01", got)
		}
		if got := r.URL.Query().Get("endDate"); got != "2024-01-31" {
			t.Errorf("endDate = %s, want 2024-01-31", got)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-01"},{"date":"2024-01-02"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetBodyBattery(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetBodyBattery: %v", err)
	}

	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

// --- GetIntensityMinutes tests ---

func TestGetIntensityMinutes_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/im/2024-01-15" {
			t.Errorf("path = %s, want /wellness-service/wellness/daily/im/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"moderateIntensityMinutes":30,"vigorousIntensityMinutes":15}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetIntensityMinutes(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetIntensityMinutes: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetWeeklyIntensityMinutes tests ---

func TestGetWeeklyIntensityMinutes_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/stats/im/weekly/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /usersummary-service/stats/im/weekly/2024-01-01/2024-01-31", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"week":"2024-W04","totalMinutes":150}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetWeeklyIntensityMinutes(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetWeeklyIntensityMinutes: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetTrainingReadiness tests ---

func TestGetTrainingReadiness_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/trainingreadiness/2024-01-15" {
			t.Errorf("path = %s, want /metrics-service/metrics/trainingreadiness/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"trainingReadinessScore":72}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetTrainingReadiness(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetTrainingReadiness: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["trainingReadinessScore"] != float64(72) {
		t.Errorf("trainingReadinessScore = %v, want 72", result["trainingReadinessScore"])
	}
}

// --- GetTrainingStatus tests ---

func TestGetTrainingStatus_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/trainingstatus/aggregated/2024-01-15" {
			t.Errorf("path = %s, want /metrics-service/metrics/trainingstatus/aggregated/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"currentTrainingLoad":850}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetTrainingStatus(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetTrainingStatus: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetFitnessAge tests ---

func TestGetFitnessAge_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fitnessage-service/fitnessage/2024-01-15" {
			t.Errorf("path = %s, want /fitnessage-service/fitnessage/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"chronologicalAge":35,"fitnessAge":28}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetFitnessAge(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetFitnessAge: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["fitnessAge"] != float64(28) {
		t.Errorf("fitnessAge = %v, want 28", result["fitnessAge"])
	}
}

// --- GetMaxMetrics tests ---

func TestGetMaxMetrics_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/maxmet/daily/2024-01-15/2024-01-15" {
			t.Errorf("path = %s, want /metrics-service/metrics/maxmet/daily/2024-01-15/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"maxMetRunning":{"vo2MaxValue":52.0}}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetMaxMetrics(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetMaxMetrics: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetLactateThreshold tests ---

func TestGetLactateThreshold_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/biometric-service/biometric/latestLactateThreshold" {
			t.Errorf("path = %s, want /biometric-service/biometric/latestLactateThreshold", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"lactateThresholdHeartRate":162}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetLactateThreshold(context.Background())
	if err != nil {
		t.Fatalf("GetLactateThreshold: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["lactateThresholdHeartRate"] != float64(162) {
		t.Errorf("lactateThresholdHeartRate = %v, want 162", result["lactateThresholdHeartRate"])
	}
}

// --- GetCyclingFTP tests ---

func TestGetCyclingFTP_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/biometric-service/biometric/latestFunctionalThresholdPower/CYCLING" {
			t.Errorf("path = %s, want /biometric-service/biometric/latestFunctionalThresholdPower/CYCLING", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"functionalThresholdPower":250}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetCyclingFTP(context.Background())
	if err != nil {
		t.Fatalf("GetCyclingFTP: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["functionalThresholdPower"] != float64(250) {
		t.Errorf("functionalThresholdPower = %v, want 250", result["functionalThresholdPower"])
	}
}

// --- GetRacePredictions tests ---

func TestGetRacePredictions_Current(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/racepredictions" {
			t.Errorf("path = %s, want /metrics-service/metrics/racepredictions", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"5k":1200,"10k":2500,"halfMarathon":5400,"marathon":11200}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRacePredictions(context.Background(), "", "")
	if err != nil {
		t.Fatalf("GetRacePredictions: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

func TestGetRacePredictions_DateRange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/racepredictions/range/2024-01-01/2024-01-31" {
			t.Errorf("path = %s, want /metrics-service/metrics/racepredictions/range/2024-01-01/2024-01-31", r.URL.Path)
		}
		if got := r.URL.Query().Get("type"); got != "daily" {
			t.Errorf("type = %s, want daily", got)
		}
		_, _ = w.Write([]byte(`[{"date":"2024-01-01"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetRacePredictions(context.Background(), "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("GetRacePredictions: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetEnduranceScore tests ---

func TestGetEnduranceScore_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/endurancescore" {
			t.Errorf("path = %s, want /metrics-service/metrics/endurancescore", r.URL.Path)
		}
		if got := r.URL.Query().Get("calendarDate"); got != "2024-01-15" {
			t.Errorf("calendarDate = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`{"overallScore":75}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetEnduranceScore(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetEnduranceScore: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["overallScore"] != float64(75) {
		t.Errorf("overallScore = %v, want 75", result["overallScore"])
	}
}

// --- GetHillScore tests ---

func TestGetHillScore_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/hillscore" {
			t.Errorf("path = %s, want /metrics-service/metrics/hillscore", r.URL.Path)
		}
		if got := r.URL.Query().Get("calendarDate"); got != "2024-01-15" {
			t.Errorf("calendarDate = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`{"overallScore":68}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetHillScore(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetHillScore: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["overallScore"] != float64(68) {
		t.Errorf("overallScore = %v, want 68", result["overallScore"])
	}
}

// --- GetAllDayEvents tests ---

func TestGetAllDayEvents_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailyEvents" {
			t.Errorf("path = %s, want /wellness-service/wellness/dailyEvents", r.URL.Path)
		}
		if got := r.URL.Query().Get("calendarDate"); got != "2024-01-15" {
			t.Errorf("calendarDate = %s, want 2024-01-15", got)
		}
		_, _ = w.Write([]byte(`[{"eventType":"activity","startGMT":"2024-01-15T08:00:00"}]`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetAllDayEvents(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetAllDayEvents: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

// --- GetLifestyleLogging tests ---

func TestGetLifestyleLogging_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lifestylelogging-service/dailyLog/2024-01-15" {
			t.Errorf("path = %s, want /lifestylelogging-service/dailyLog/2024-01-15", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"calendarDate":"2024-01-15","entries":[]}`))
	})

	_, client := testServer(t, handler)
	data, err := client.GetLifestyleLogging(context.Background(), "2024-01-15")
	if err != nil {
		t.Fatalf("GetLifestyleLogging: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
}

func TestGetLifestyleLogging_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.GetLifestyleLogging(context.Background(), "2024-01-15")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
}
