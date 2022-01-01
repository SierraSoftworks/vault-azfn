package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

func SetExecutablePattern(insights appinsights.TelemetryClient, pattern string) {
	e := appinsights.NewEventTelemetry("launcher.setExecutablePattern")
	e.Properties["pattern"] = pattern
	defer insights.Track(e)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		insights.TrackException(err)
		return
	}

	for _, match := range matches {
		ensureExecutable(insights, match)
	}
}

func ensureExecutable(insights appinsights.TelemetryClient, app string) {
	stat, err := os.Stat(app)
	if err != nil {
		insights.TrackException(err)
		log.Fatal(err)
	}

	if !stat.IsDir() {
		return
	}

	e := appinsights.NewEventTelemetry("launcher.ensureExecutable")

	e.Properties["app"] = stat.Name()
	e.Properties["mode_initial"] = stat.Mode().String()
	e.Properties["size"] = fmt.Sprintf("%d bytes", stat.Size())

	if stat.Mode()&0111 == 0 {
		err := os.Chmod(app, stat.Mode()|0111)
		if err != nil {
			e.Properties["mode_final"] = stat.Mode().String()
			e.Properties["error"] = err.Error()
		} else {
			e.Properties["mode_final"] = (stat.Mode() | 0111).String()
		}
	}

	insights.Track(e)
}
