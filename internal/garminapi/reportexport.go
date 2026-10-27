package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var reportExportEndpoints = []Endpoint{
	{
		Name: "reportexport.init", Method: http.MethodPost, Host: ConnectAPI, Read: true, Body: true,
		Path:   "/reportexport-service/export/init/{category}/{start}/{end}?locale={locale}",
		Params: []Param{{Name: "category", Enum: []string{reportBloodPressure, reportHealthSnapshot}}},
	},
	{Name: "reportexport.status", Method: http.MethodGet, Host: ConnectAPI, Path: "/reportexport-service/export/reportStatus/{id}"},
}

const (
	reportBloodPressure  = "bloodPressure"
	reportHealthSnapshot = "wellnessactivity"
)

// ReportPeriod is the layout of a blood pressure report, as Garmin names it.
// It also fixes the range: Garmin answers DATA_NOT_FOUND when the dates do
// not span Days() days, even if readings exist.
type ReportPeriod string

const (
	PeriodDaily     ReportPeriod = "Daily"
	PeriodSevenDays ReportPeriod = "SevenDays"
	PeriodFourWeeks ReportPeriod = "FourWeeks"
	PeriodYearly    ReportPeriod = "Yearly"
)

// Days is the length of the range that the period covers, end date included.
func (p ReportPeriod) Days() int {
	switch p {
	case PeriodSevenDays:
		return 7
	case PeriodFourWeeks:
		return 28
	case PeriodYearly:
		return 364
	default:
		return 1
	}
}

// ReportPreferences are the profile settings the web app puts in a report
// request: the language of the report and the date format of its tables.
type ReportPreferences struct {
	Locale        string
	DateFormatKey string
}

// GetReportPreferences reads preferredLocale and dateFormat.formatKey from
// the profile settings. A profile without a locale gets "en". A profile
// without a date format key is an error, because Garmin needs one for a
// blood pressure report.
func (c *Client) GetReportPreferences(ctx context.Context) (ReportPreferences, error) {
	data, err := c.Call(ctx, "profile.get", nil, nil)
	if err != nil {
		return ReportPreferences{}, err
	}
	var settings struct {
		PreferredLocale string `json:"preferredLocale"`
		DateFormat      struct {
			FormatKey string `json:"formatKey"`
		} `json:"dateFormat"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return ReportPreferences{}, fmt.Errorf("parse profile settings: %w", err)
	}
	if settings.DateFormat.FormatKey == "" {
		return ReportPreferences{}, fmt.Errorf("profile settings missing dateFormat.formatKey")
	}
	p := ReportPreferences{Locale: settings.PreferredLocale, DateFormatKey: settings.DateFormat.FormatKey}
	if p.Locale == "" {
		p.Locale = defaultReportLocale
	}
	return p, nil
}

// ReportRequest is the init request of one report export: the category, the
// date range and the locale of the request, and the category's body.
type ReportRequest struct {
	category   string
	start, end string
	locale     string
	body       map[string]any
}

// BloodPressureReport asks for the blood pressure report of a date range.
func BloodPressureReport(start, end string, period ReportPeriod, prefs ReportPreferences) ReportRequest {
	return ReportRequest{
		category: reportBloodPressure, start: start, end: end, locale: prefs.Locale,
		body: map[string]any{
			"contentToSkip":          []string{},
			"reportSubCategory":      period,
			"reportFormat":           "PDF",
			"requestedDateFormatKey": prefs.DateFormatKey,
		},
	}
}

// HealthSnapshotReport asks for the PDF report of a Health Snapshot session.
// date is the calendar date of the session.
func HealthSnapshotReport(date, uuid string, prefs ReportPreferences) ReportRequest {
	return ReportRequest{
		category: reportHealthSnapshot, start: date, end: date, locale: prefs.Locale,
		body: map[string]any{"supplementalId": uuid},
	}
}

const (
	defaultReportLocale = "en"
	reportDefaultWait   = 3 * time.Second
	reportMinWait       = time.Second
	reportMaxWait       = 10 * time.Second
	reportMaxTotalWait  = 2 * time.Minute
	reportReady         = "READY"
	reportCreating      = "CREATING"
)

// ReportUnavailableError means Garmin ended the export in a state other than
// READY, such as DATA_NOT_FOUND for a range without readings.
type ReportUnavailableError struct {
	Status  string
	Message string
}

func (e *ReportUnavailableError) Error() string {
	msg := "Garmin did not create the report: status " + e.Status
	if e.Status == "DATA_NOT_FOUND" {
		msg = "Garmin has no data for this report (status DATA_NOT_FOUND)"
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

// ReportTimeoutError means the report was still being created after the
// client had waited as long as it is willing to.
type ReportTimeoutError struct {
	Waited time.Duration
}

func (e *ReportTimeoutError) Error() string {
	return fmt.Sprintf("Garmin was still creating the report after %s", e.Waited)
}

// ExportReport has Garmin create a report and returns the file. It starts
// the export, polls its status every estimatedTimeForStatusCheck (kept
// between one and ten seconds) until READY, and downloads the content. Any
// other final status is a *ReportUnavailableError, and a report that is still
// CREATING after two minutes of waiting is a *ReportTimeoutError.
func (c *Client) ExportReport(ctx context.Context, req ReportRequest) ([]byte, error) {
	body, err := json.Marshal(req.body)
	if err != nil {
		return nil, fmt.Errorf("marshal report request: %w", err)
	}
	args := map[string]string{"category": req.category, "start": req.start, "end": req.end, "locale": req.locale}
	raw, err := c.Call(ctx, "reportexport.init", args, body)
	if err != nil {
		return nil, err
	}
	var job struct {
		ReportID      string `json:"reportId"`
		EstimatedTime int    `json:"estimatedTimeForStatusCheck"`
	}
	if err := json.Unmarshal(raw, &job); err != nil || job.ReportID == "" {
		return nil, fmt.Errorf("report export: Garmin answered without a reportId: %.200q", raw)
	}

	wait := pollInterval(job.EstimatedTime)
	for waited := time.Duration(0); ; {
		if waited >= reportMaxTotalWait {
			return nil, &ReportTimeoutError{Waited: waited}
		}
		if err := c.reportWait(ctx, wait); err != nil {
			return nil, err
		}
		waited += wait

		raw, err := c.Call(ctx, "reportexport.status", map[string]string{"id": job.ReportID}, nil)
		if err != nil {
			return nil, err
		}
		var st struct {
			Progress struct {
				Status          string `json:"status"`
				ResponseMessage string `json:"responseMessage"`
			} `json:"progress"`
		}
		if err := json.Unmarshal(raw, &st); err != nil {
			return nil, fmt.Errorf("report export: parse status: %w", err)
		}
		switch st.Progress.Status {
		case reportReady:
			return c.Download(ctx, "/reportexport-service/export/reportContent/"+url.PathEscape(job.ReportID))
		case reportCreating:
		default:
			return nil, &ReportUnavailableError{Status: st.Progress.Status, Message: st.Progress.ResponseMessage}
		}
	}
}

func pollInterval(estimatedMillis int) time.Duration {
	if estimatedMillis <= 0 {
		return reportDefaultWait
	}
	return min(max(time.Duration(estimatedMillis)*time.Millisecond, reportMinWait), reportMaxWait)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
