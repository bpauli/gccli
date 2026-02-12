package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

// ActivityDownloadCmd downloads an activity in a specified format.
type ActivityDownloadCmd struct {
	ID     string `arg:"" help:"Activity ID."`
	Format string `help:"Download format: fit, gpx, tcx, kml, csv." default:"fit" enum:"fit,gpx,tcx,kml,csv"`
	Output string `help:"Output file path (default: activity_{id}.{format})." short:"o"`
}

func (c *ActivityDownloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	format := garminapi.ActivityDownloadFormat(strings.ToLower(c.Format))

	data, err := client.DownloadActivity(g.Context, c.ID, format)
	if err != nil {
		return fmt.Errorf("download activity: %w", err)
	}

	// FIT downloads come as a zip archive; extract the .fit file.
	if format == garminapi.FormatFIT {
		extracted, err := extractFIT(data)
		if err != nil {
			return fmt.Errorf("extract FIT from zip: %w", err)
		}
		data = extracted
	}

	outPath := c.Output
	if outPath == "" {
		outPath = defaultActivityFilename(c.ID, c.Format)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	g.UI.Successf("Downloaded %s (%d bytes)", outPath, len(data))
	return nil
}

// defaultActivityFilename returns the default output filename for a download.
func defaultActivityFilename(id, format string) string {
	return fmt.Sprintf("activity_%s.%s", id, format)
}

// extractFIT extracts the first .fit file from a zip archive.
func extractFIT(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	for _, f := range r.File {
		if strings.EqualFold(filepath.Ext(f.Name), ".fit") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open %s in zip: %w", f.Name, err)
			}
			defer func() { _ = rc.Close() }()

			var buf bytes.Buffer
			if _, err := buf.ReadFrom(rc); err != nil {
				return nil, fmt.Errorf("read %s from zip: %w", f.Name, err)
			}
			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("no .fit file found in zip archive")
}
