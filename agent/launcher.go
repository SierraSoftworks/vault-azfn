package agent

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

func RunApp(insights appinsights.TelemetryClient, app string, args []string) error {
	ensureExecutable(insights, app)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command(app, args...)
	cmd.Stdout = NewInsightsLogStream(insights)
	cmd.Stderr = NewInsightsLogStream(insights)
	cmd.Env = os.Environ()
	cmd.Dir = cwd

	c := make(chan os.Signal, 1)
	signal.Notify(c)

	exit := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case s := <-c:
				if cmd.Process != nil {
					cmd.Process.Signal(s)
				}
			case <-exit:
				return
			}
		}
	}()

	e := appinsights.NewEventTelemetry("launcher.run.startApp")
	e.Properties["app"] = app
	e.Properties["args"] = strings.Join(args, " ")
	e.Properties["cwd"] = cwd
	insights.Track(e)

	err = cmd.Run()

	e.Name = "launcher.run.exit"
	e.Properties["status"] = err.Error()
	insights.Track(e)
	exit <- struct{}{}

	return err
}
