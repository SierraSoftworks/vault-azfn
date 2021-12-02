package agent

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

func ensureExecutable(insights appinsights.TelemetryClient, app string) {
	stat, err := os.Stat(app)
	if err != nil {
		log.Fatal(err)
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
