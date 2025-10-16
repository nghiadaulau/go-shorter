package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Options struct {
	Endpoint string
	Service  string
}

func Setup(ctx context.Context, opts Options) (func(context.Context) error, error) {
	client := otlptracehttp.NewClient(otlptracehttp.WithEndpoint(opts.Endpoint), otlptracehttp.WithInsecure())
	exp, err := otlptrace.New(ctx, client)
	if err != nil { return nil, err }
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.Empty()),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
