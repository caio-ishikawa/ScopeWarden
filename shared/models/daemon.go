package models

import (
	"time"
)

type DaemonStats struct {
	// Represents the total number of found URLs in this current scan
	TotalFoundURLs int
	// Represents all new URLs (includes URLs that already existed but were down previously)
	TotalNewURLs int
	// Represents the total number of port found in this current scan
	TotalFoundPorts int
	// Represents all new port founds in this current scan (includes ports that were closed/filtered and are now open)
	TotalNewPorts int
	// Represents how long the current scan has been running
	ScanTime time.Duration
	// Represents the time the current scan began
	ScanBegin time.Time
	// Represents the time where the last scan ended
	LastScanEnded time.Time
	// Represents whether or not the scan is running currently
	IsRunning bool
}
