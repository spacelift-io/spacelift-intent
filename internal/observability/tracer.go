package observability

import (
	"log"
	"strings"

	"github.com/aws/aws-xray-sdk-go/strategy/ctxmissing"
	libxray "github.com/aws/aws-xray-sdk-go/xray"
	"github.com/aws/aws-xray-sdk-go/xraylog"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spcontext"
	"github.com/spacelift-io/spcontext/tracing/datadog"
	spOtel "github.com/spacelift-io/spcontext/tracing/opentelemetry"
	"github.com/spacelift-io/spcontext/tracing/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	otelTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Vendor represents a vendor of observability services. We generally support
// two: Datadog and AWS.
type Vendor string

const (
	// VendorDisabled is a special vendor that disables tracing and metrics.
	VendorDisabled Vendor = "Disabled"

	// VendorAWS is the AWS vendor, using CloudWatch Logs for logs, CloudWatch
	// metrics for metrics and X-Ray for tracing.
	VendorAWS Vendor = "AWS"

	// VendorDatadog is the Datadog vendor, using Datadog
	// for everything (tracing, metrics).
	VendorDatadog Vendor = "Datadog"

	// VendorOpenTelemetry is the OpenTelemetry vendor, using OpenTelemetry
	// for everything (tracing, metrics).
	VendorOpenTelemetry Vendor = "OpenTelemetry"
)

func ParseVendor(vendor string) (Vendor, error) {
	switch strings.ToLower(vendor) {
	case "disabled":
		return VendorDisabled, nil
	case "aws":
		return VendorAWS, nil
	case "datadog":
		return VendorDatadog, nil
	case "opentelemetry":
		return VendorOpenTelemetry, nil
	default:
		return "", errors.Errorf("unknown observability vendor: %s", vendor)
	}
}

// NewTracer returns a tracer for the current vendor, and a function to start and
// stop it. Opts are only used for Datadog, since there is nothing worth
// customizing in the X-Ray tracer implementation.
// TODO: Add git tags to the tracer
func NewTracer(ctx *spcontext.Context, serviceName string, vendor Vendor, ddOpts ...tracer.StartOption) (_ spcontext.Tracer, start func() (stop func())) {
	if vendor == VendorDisabled {
		return &spcontext.NopTracer{}, func() func() {
			return func() {}
		}
	}

	if vendor == VendorDatadog {
		ddOpts = append(ddOpts, tracer.WithService(serviceName))

		return &datadog.Tracer{}, func() func() {
			tracer.Start(ddOpts...)
			return tracer.Stop
		}
	}

	if vendor == VendorAWS {
		return &xray.Tracer{}, func() func() {
			libxray.SetLogger(NewXrayLogger(ctx, xraylog.LogLevelWarn))
			libxray.Configure(libxray.Config{
				ContextMissingStrategy: &ctxmissing.DefaultIgnoreErrorStrategy{},
			})

			return func() {}
		}
	}

	if vendor == VendorOpenTelemetry {
		r, err := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
			),
		)
		if err != nil {
			log.Panic(errors.Wrap(err, "failed to create OTEL resource"))
		}

		traceExporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			log.Panic(errors.Wrap(err, "failed to create OTEL trace exporter"))
		}

		tp := otelTrace.NewTracerProvider(
			otelTrace.WithBatcher(traceExporter),
			otelTrace.WithResource(r),
		)

		metricExporter, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			log.Panic(errors.Wrap(err, "failed to create OTEL metric exporter"))
		}

		mp := metric.NewMeterProvider(
			metric.WithResource(r),
			metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		)

		otel.SetTracerProvider(tp)
		otel.SetMeterProvider(mp)

		return &spOtel.Tracer{}, func() (stop func()) {
			return func() {
				_ = tp.Shutdown(ctx)
				_ = mp.Shutdown(ctx)
			}
		}
	}

	panic("unknown observability vendor")
}
