package agent

import (
	"context"
	"encoding/json"
	"os"

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
	_, span := otel.Tracer("vault").Start(s.ctx, "launcher.TelemetryLogStream.Write", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	props := map[string]string{}
	if err := json.Unmarshal(p, &props); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.span.AddEvent(
			"Failed to parse log message",
			trace.WithAttributes(attribute.String("message", string(p)), attribute.String("error", err.Error())),
			trace.WithStackTrace(true))

		return os.Stdout.Write(p)
	}

	properties := []attribute.KeyValue{}
	for k, v := range props {
		properties = append(properties, attribute.String(k, v))
	}

	span.SetAttributes(properties...)
	span.SetName(props["@message"])

	return len(p), nil
}
