package agent

import (
	"context"
	"os"
	"os/exec"
	"os/signal"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RunApp(ctx context.Context, app string, args []string) error {
	ctx, span := otel.Tracer("vault").Start(ctx, "launcher.RunApp", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ensureExecutable(ctx, app)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command(app, args...)
	cmd.Stdout = NewTelemetryLogStream(ctx, span)
	cmd.Stderr = NewTelemetryLogStream(ctx, span)
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
					span.AddEvent("Propagating signal to child process.", trace.WithAttributes(attribute.String("signal", s.String())))
					cmd.Process.Signal(s)
				}
			case <-exit:
				return
			}
		}
	}()

	span.AddEvent("Starting Vault", trace.WithAttributes(attribute.String("app", app), attribute.StringSlice("args", args), attribute.String("cwd", cwd)))

	err = cmd.Run()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	exit <- struct{}{}

	return err
}
