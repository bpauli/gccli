package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// exercisesURL is the public endpoint for the Garmin exercise catalog.
const exercisesURL = "https://connect.garmin.com/web-data/exercises/Exercises.json"

// fetchExercisesURL is a variable for testing.
var fetchExercisesURL = exercisesURL

// FetchExerciseCatalog fetches the public Garmin exercise catalog.
// This does not require authentication.
func FetchExerciseCatalog(ctx context.Context) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchExercisesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch exercise catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch exercise catalog: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read exercise catalog: %w", err)
	}

	return json.RawMessage(data), nil
}
