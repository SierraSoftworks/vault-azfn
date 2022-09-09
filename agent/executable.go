package agent

import (
	"context"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func SetExecutablePattern(ctx context.Context, pattern string) {
	ctx, span := otel.Tracer("vault").Start(ctx, "launcher.SetExecutablePattern")
	defer span.End()

	span.SetAttributes(attribute.String("pattern", pattern))

	matches, err := filepath.Glob(pattern)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	for _, match := range matches {
		ensureExecutable(ctx, match)
	}
}

func ensureExecutable(ctx context.Context, app string) {
	_, span := otel.Tracer("vault").Start(ctx, "launcher.ensureExecutable")
	defer span.End()

	span.SetAttributes(attribute.String("app", app))

	stat, err := os.Stat(app)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	if stat.IsDir() {
		return
	}

	span.SetAttributes(
		attribute.String("mode.initial", stat.Mode().String()),
		attribute.String("app", stat.Name()),
		attribute.Int64("size", stat.Size()))

	if stat.Mode()&0111 == 0 {
		err := os.Chmod(app, stat.Mode()|0111)
		if err != nil {
			span.SetAttributes(attribute.String("mode.final", stat.Mode().String()))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetAttributes(attribute.String("mode.final", (stat.Mode() | 0111).String()))
		}
	}
}
