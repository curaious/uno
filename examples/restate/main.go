package main

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/restatedev/sdk-go/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	tracer = otel.Tracer("Example")
)

type Input struct{}

type Output struct{}

type Workflow struct{}

func (w Workflow) Run(restateCtx restate.WorkflowContext, input Input) (Output, error) {
	ctx, span := tracer.Start(restateCtx, "Workflow.Run")
	defer span.End()

	restate.Run(restate.WrapContext(restateCtx, ctx), func(runCtx restate.RunContext) (any, error) {
		_, span := tracer.Start(runCtx, "SomeWork")
		defer span.End()

		return w.SomeWork(runCtx), nil
	})

	return Output{}, nil
}

func (w Workflow) SomeWork(ctx context.Context) any {
	return nil
}

func main() {
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

	go func() {
		if err := server.NewRestate().
			Bind(restate.Reflect(Workflow{})).
			Start(context.Background(), "0.0.0.0:8065"); err != nil {
			log.Fatal(err)
		}
	}()

	http.ListenAndServe(":8066", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "RestAPI")
		defer span.End()

		restateClient := ingress.NewClient("http://localhost:8081")
		_, err = ingress.Workflow[*Input, *Output](
			restateClient,
			"Workflow",
			uuid.NewString(),
			"Run",
		).Request(ctx, &Input{})
		if err != nil {
			log.Fatal(err)
			return
		}
	}))
}
