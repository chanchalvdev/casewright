package classifier

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/chanchalvdev/casewright/casewright/internal/models"
	"github.com/chanchalvdev/casewright/casewright/internal/tracing"
)

// The global tracer delegates to whatever provider tracing.Setup installs,
// and is a no-op when tracing is off.
var tracer = otel.Tracer("github.com/chanchalvdev/casewright/casewright/internal/classifier")

// maxTraceContent caps the log line and answer stored on a span: adversarial
// inputs can be very long.
const maxTraceContent = 4096

// recordSpan describes one LLM call. Attribute names follow the OpenTelemetry
// GenAI conventions plus the input.value/output.value pair that both Langfuse
// and Phoenix display; casewright.* holds the classifier's own verdict.
func (c *Classifier) recordSpan(span trace.Span, call llmCall, result models.ClassifiedEvent, err error) {
	if !span.IsRecording() {
		return
	}
	env, version := tracing.Env(), c.PromptVersion()
	attrs := []attribute.KeyValue{
		attribute.String("gen_ai.operation.name", "chat"),
		attribute.String("gen_ai.request.model", c.model),
		attribute.Float64("gen_ai.request.temperature", 0.1),
		attribute.String("openinference.span.kind", "LLM"),
		attribute.String("casewright.prompt_version", version),
		attribute.String("casewright.env", env),
		attribute.String("langfuse.environment", env),
		attribute.String("langfuse.trace.name", "classify"),
		attribute.StringSlice("langfuse.trace.tags", []string{"env:" + env, "prompt:" + version}),
	}
	if tracing.ContentEnabled() {
		attrs = append(attrs,
			attribute.String("input.value", clip(call.input)),
			attribute.String("output.value", clip(call.output)),
		)
	}
	if u := call.usage; u != nil {
		attrs = append(attrs,
			attribute.Int("gen_ai.usage.input_tokens", u.PromptTokens),
			attribute.Int("gen_ai.usage.output_tokens", u.CompletionTokens),
		)
	}
	if err != nil {
		span.SetAttributes(append(attrs, attribute.Bool("casewright.valid", false))...)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetAttributes(append(attrs,
		attribute.String("casewright.attack_type", result.AttackType),
		attribute.String("casewright.severity", string(result.Severity)),
		attribute.Float64("casewright.confidence", result.Confidence),
		attribute.Bool("casewright.valid", len(result.OutputIssues) == 0),
		attribute.StringSlice("casewright.output_issues", result.OutputIssues),
	)...)
}

func clip(s string) string {
	if len(s) <= maxTraceContent {
		return s
	}
	return s[:maxTraceContent] + "…[truncated]"
}
