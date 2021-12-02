package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
)

func GetInsights() appinsights.TelemetryClient {
	ik := os.Getenv("APPINSIGHTS_INSTRUMENTATIONKEY")
	enabled := ik != ""
	if ik == "" {
		ik = "NULL"
	}

	insights := appinsights.NewTelemetryClient(ik)
	insights.SetIsEnabled(enabled)

	return insights
}

type InsightsLogStream struct {
	insights appinsights.TelemetryClient
	in       io.Writer
	scanner  *bufio.Scanner
}

func NewInsightsLogStream(insights appinsights.TelemetryClient) *InsightsLogStream {
	buffer := bytes.NewBuffer([]byte{})
	scanner := bufio.NewScanner(buffer)
	scanner.Split(bufio.ScanLines)

	return &InsightsLogStream{
		insights: insights,
		in:       buffer,
		scanner:  scanner,
	}
}

func (s *InsightsLogStream) Write(p []byte) (n int, err error) {
	s.in.Write(p)

	for s.scanner.Scan() {
		props := map[string]string{}
		if err := json.Unmarshal([]byte(s.scanner.Text()), &props); err != nil {
			s.insights.TrackTrace(s.scanner.Text(), contracts.Information)
		} else {
			t := appinsights.NewTraceTelemetry(props["@message"], getSeverityLevel(props["@level"]))
			t.Properties = props
			if ts, err := time.Parse(time.RFC3339, props["@timestamp"]); err == nil {
				t.Timestamp = ts
			}

			s.insights.Track(t)
		}
	}

	return len(p), nil
}

func getSeverityLevel(level string) contracts.SeverityLevel {
	switch level {
	case "debug":
		return contracts.Verbose
	case "info":
		return contracts.Information
	case "warning":
		return contracts.Warning
	case "error":
		return contracts.Error
	default:
		return contracts.Information
	}
}
