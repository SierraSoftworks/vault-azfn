package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

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
	if !strings.HasPrefix(string(p), `{"`) {
		s.span.AddEvent("log", trace.WithAttributes(attribute.String("@message", string(p))))
	} else {
		for _, line := range strings.Split(strings.TrimSpace(string(p)), "\n") {
			line := strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if err := s.WriteMessage(line); err != nil {
				s.span.RecordError(err, trace.WithAttributes(attribute.String("@message", line)))
			}
		}
	}

	return os.Stdout.Write(p)
}

func (s *TelemetryLogStream) WriteMessage(msg string) error {
	props := map[string]interface{}{}
	if err := json.Unmarshal([]byte(msg), &props); err != nil {
		return err
	}

	timestamp := getSpanStartTime(props)

	_, span := otel.Tracer("vault").Start(
		s.ctx,
		getSpanName(props),
		trace.WithSpanKind(getSpanKind(props)),
		trace.WithAttributes(attribute.String("@message", msg)),
		trace.WithTimestamp(timestamp),
		trace.WithLinks(trace.Link{SpanContext: s.span.SpanContext()}),
		trace.WithNewRoot(),
	)

	setSpanPropertiesAndEnd(span, timestamp, props)

	return nil
}

func getSpanName(props map[string]interface{}) string {
	name := "log"
	switch props["@module"] {
	case nil:
	default:
		name = toString(props["@module"])
	}

	return name
}

func getSpanKind(props map[string]interface{}) trace.SpanKind {
	if props["@module"] == "core" && props["@message"] == "completed_request" {
		return trace.SpanKindServer
	}

	return trace.SpanKindInternal
}

func getSpanStartTime(props map[string]interface{}) time.Time {
	timestamp := time.Now()
	switch tss := props["@timestamp"].(type) {
	case string:
		ts, err := time.Parse("2006-01-02T15:04:05.000000Z", tss)
		if err == nil {
			timestamp = ts
		}
	default:
	}

	return timestamp
}

func setSpanPropertiesAndEnd(span trace.Span, startTime time.Time, props map[string]interface{}) {
	endTime := startTime
	defer func() { span.End(trace.WithTimestamp(endTime)) }()

	properties := []attribute.KeyValue{}
	for k, v := range props {
		if k == "error" {
			span.SetStatus(codes.Error, toString(v))
			continue
		}

		if k == "@level" && v == "error" {
			span.SetStatus(codes.Error, toString(props["@message"]))
			continue
		}

		if k == "@timestamp" {
			continue
		}

		if k == "duration" {
			ds := v.(string)
			d, err := time.ParseDuration(ds)
			if err == nil {
				endTime = startTime.Add(d)
				continue
			}
		}

		switch v := v.(type) {
		case string:
			properties = append(properties, attribute.String(k, v))
		case int:
			properties = append(properties, attribute.Int(k, v))
		case float64:
			properties = append(properties, attribute.Float64(k, v))
		case bool:
			properties = append(properties, attribute.Bool(k, v))
		default:
			properties = append(properties, attribute.String(k, toJsonString(v)))
		}
	}

	span.SetAttributes(properties...)

	span.End(trace.WithTimestamp(endTime))
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return toJsonString(v)
	}
}

func toJsonString(v interface{}) string {
	out := bytes.NewBufferString("")
	json.NewEncoder(out).Encode(v)
	return out.String()
}
