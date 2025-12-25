package main

import (
	"context"
	"log"

	_ "github.com/lib/pq"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	trace2 "go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("Workflow")
)

type Input struct{}

type Output struct{}

type Workflow struct{}

func (w Workflow) Run(restateCtx restate.Context, input Input) (Output, error) {
	ctx, span := tracer.Start(restateCtx, "Workflow.Run")
	defer span.End()
	//restate.WrapRestateContext(restateCtx, ctx)
	restateCtx = restate.WrapContext(restateCtx, ctx)
	restate.Run(restateCtx, func(runCtx restate.RunContext) (any, error) {
		_, span := tracer.Start(restateCtx, "SomeWork")
		defer span.End()

		return w.SomeWork(trace2.ContextWithSpanContext(runCtx, span.SpanContext())), nil
	})

	return Output{}, nil
}

func (w Workflow) SomeWork(ctx context.Context) any {
	//ctx, span := tracer.Start(ctx, "SomeWork")
	//defer span.End()

	return nil
}

func main() {
	//cmd.Execute()
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("localhost:4318"),
	)
	if err != nil {
		log.Fatal(err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "restate-sdk-go-otel-example-greeter"),
		),
	)
	if err != nil {
		log.Fatalf("Could not set resources: %v", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
		trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exporter)),
		trace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	if err := server.NewRestate().
		Bind(restate.Reflect(Workflow{})).
		Start(context.Background(), "0.0.0.0:8065"); err != nil {
		log.Fatal(err)
	}

}
