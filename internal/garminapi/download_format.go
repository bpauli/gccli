package garminapi

// DownloadFormat represents a supported export format.
type DownloadFormat string

const (
	FormatFIT DownloadFormat = "fit"
	FormatTCX DownloadFormat = "tcx"
	FormatGPX DownloadFormat = "gpx"
	FormatKML DownloadFormat = "kml"
	FormatCSV DownloadFormat = "csv"
)
