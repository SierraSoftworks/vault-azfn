package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

func NewTraceProvider(ctx context.Context, version string) *sdktrace.TracerProvider {
	resource, err := resource.Merge(
		resource.Environment(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("vault"),
			semconv.ServiceNamespaceKey.String("github.com/SierraSoftworks/vault-azfn"),
			semconv.ServiceVersionKey.String(version),
		))

	if err != nil {
		panic(err)
	}

	exporter, err := newExporter(ctx)
	if err != nil {
		panic(err)
	}

	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	return tracer
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	client := otlptracegrpc.NewClient()
	return otlptrace.New(ctx, client)
}

type TelemetryLogStream struct {
	ctx  context.Context
	span trace.Span
}

func NewTelemetryLogStream(ctx context.Context, span trace.Span) *TelemetryLogStream {
	return &TelemetryLogStream{
		ctx,
		span,
	}
}

func (s *TelemetryLogStream) Write(p []byte) (n int, err error) {
	for _, line := range strings.Split(string(p), "\n") {
		if err := s.WriteMessage(line); err != nil {
			s.span.RecordError(err, trace.WithAttributes(attribute.String("raw_message", line)))
		}
	}

	return os.Stdout.Write(p)
}

func (s *TelemetryLogStream) WriteMessage(msg string) error {
	_, span := otel.Tracer("vault").Start(s.ctx, "launcher.TelemetryLogStream.Write", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	span.SetAttributes(attribute.String("raw_message", msg))

	props := map[string]string{}
	if err := json.Unmarshal([]byte(msg), &props); err != nil {
		span.SetName(msg)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	properties := []attribute.KeyValue{}
	for k, v := range props {
		properties = append(properties, attribute.String(k, v))
	}

	span.SetAttributes(properties...)
	span.SetName(props["@message"])

	return nil
}
