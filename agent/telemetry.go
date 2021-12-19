package agent

import (
	"encoding/json"
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

	insights.Context().CommonProperties["Category"] = "Vault"

	return insights
}

type InsightsLogStream struct {
	insights appinsights.TelemetryClient
}

func NewInsightsLogStream(insights appinsights.TelemetryClient) *InsightsLogStream {
	return &InsightsLogStream{
		insights: insights,
	}
}

func (s *InsightsLogStream) Write(p []byte) (n int, err error) {
	props := map[string]string{}
	if err := json.Unmarshal(p, &props); err != nil {
		return os.Stdout.Write(p)
	}

	t := appinsights.NewTraceTelemetry(props["@message"], getSeverityLevel(props["@level"]))
	t.Properties = props
	if ts, err := time.Parse(time.RFC3339, props["@timestamp"]); err == nil {
		t.Timestamp = ts
	}

	s.insights.Track(t)

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
