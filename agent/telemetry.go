package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

func NewTraceProvider(ctx context.Context, version string) (*sdktrace.TracerProvider, *log.LoggerProvider) {
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

	loggerProvider, err := newLoggerProvider(ctx, resource)
	if err != nil {
		panic(err)
	}

	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	hook := otellogrus.NewHook("github.com/sierrasoftworks/vault-azfn", otellogrus.WithLoggerProvider(loggerProvider))
	logrus.AddHook(hook)

	return tracer, loggerProvider
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	client := otlptracegrpc.NewClient()
	return otlptrace.New(ctx, client)
}

func newLoggerProvider(ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	processor := log.NewBatchProcessor(exporter)
	provider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(processor),
	)
	return provider, nil
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
		logrus.Info(string(p))
	} else {
		for _, line := range strings.Split(strings.TrimSpace(string(p)), "\n") {
			line := strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if err := s.WriteMessage(line); err != nil {
				logrus.Info(line)
				logrus.WithError(err).WithField("line", line).Warn("Failed to parse telemetry log message")
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

	kind := getSpanKind(props)
	if kind == trace.SpanKindServer {
		_, span := otel.Tracer("vault").Start(
			s.ctx,
			getSpanName(props),
			trace.WithSpanKind(kind),
			trace.WithAttributes(attribute.String("@message", msg)),
			trace.WithTimestamp(timestamp),
			trace.WithLinks(trace.Link{SpanContext: s.span.SpanContext()}),
			trace.WithNewRoot(),
		)

		setSpanPropertiesAndEnd(span, timestamp, props)
	}

	logMessage(timestamp, props)

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

func logMessage(startTime time.Time, props map[string]interface{}) {
	level := props["@level"]
	msg := toString(props["@message"])

	delete(props, "@timestamp")
	delete(props, "@level")
	delete(props, "@message")

	event := logrus.WithTime(startTime).WithFields(props)
	switch level {
	case "debug":
		event.Debug(msg)
	case "info":
		event.Info(msg)
	case "warn":
		event.Warn(msg)
	case "error":
		event.Error(msg)
	default:
		event.Info(msg)
	}
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
