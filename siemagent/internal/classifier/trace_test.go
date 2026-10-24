package classifier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var (
	spanExporter     = tracetest.NewInMemoryExporter()
	spanProviderOnce sync.Once
)

// recordSpans routes spans to memory. The global tracer binds to the first
// provider ever installed, so every test shares one and resets it.
func recordSpans(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	spanProviderOnce.Do(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(spanExporter)))
	})
	spanExporter.Reset()
	return spanExporter
}

func attrs(s tracetest.SpanStub) map[attribute.Key]attribute.Value {
	m := map[attribute.Key]attribute.Value{}
	for _, kv := range s.Attributes {
		m[kv.Key] = kv.Value
	}
	return m
}

// sseServer streams the given raw SSE lines, then [DONE].
func sseServer(lines ...string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, l := range lines {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", l)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func TestClassify_RecordsSpan(t *testing.T) {
	exp := recordSpans(t)
	srv := sseServer(
		`{"id":"x","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"{\"attack_type\":\"Brute Force\",\"severity\":\"P3\",\"confidence\":0.8,\"iocs\":[]}"}}]}`,
		`{"id":"x","object":"chat.completion.chunk","model":"m","choices":[],"usage":{"prompt_tokens":120,"completion_tokens":40,"total_tokens":160}}`,
	)
	defer srv.Close()
	c := newTestClassifier(srv)
	c.SetStreamUsage(true)

	if _, err := c.Classify(context.Background(), sampleEvent()); err != nil {
		t.Fatalf("Classify: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	a := attrs(spans[0])
	checks := map[attribute.Key]string{
		"gen_ai.request.model":      "test-model",
		"casewright.attack_type":    "Brute Force",
		"casewright.prompt_version": DefaultPromptVersion,
		"casewright.env":            "live",
	}
	for k, want := range checks {
		if got := a[k].AsString(); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if !a["casewright.valid"].AsBool() {
		t.Error("casewright.valid = false, want true")
	}
	if a["gen_ai.usage.input_tokens"].AsInt64() != 120 || a["gen_ai.usage.output_tokens"].AsInt64() != 40 {
		t.Errorf("usage = %v / %v", a["gen_ai.usage.input_tokens"], a["gen_ai.usage.output_tokens"])
	}
	if !strings.Contains(a["input.value"].AsString(), "Failed password") {
		t.Errorf("input.value missing the log line: %q", a["input.value"].AsString())
	}
}

func TestClassify_BrokenStreamIsTransportError(t *testing.T) {
	exp := recordSpans(t)
	srv := sseServer(
		`{"id":"x","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"{\"attack_"}}]}`,
		`{not json`,
	)
	defer srv.Close()

	_, err := newTestClassifier(srv).Classify(context.Background(), sampleEvent())
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, ErrInvalidOutput) {
		t.Errorf("a broken stream was reported as invalid model output: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 || spans[0].Status.Code != codes.Error {
		t.Fatalf("want one error span, got %+v", spans)
	}
}

func TestClip(t *testing.T) {
	long := strings.Repeat("a", maxTraceContent+10)
	if got := clip(long); len(got) > maxTraceContent+20 || !strings.HasSuffix(got, "[truncated]") {
		t.Errorf("clip did not truncate: len=%d", len(got))
	}
	if clip("short") != "short" {
		t.Error("clip changed a short string")
	}
}
