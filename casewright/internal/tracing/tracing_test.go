package tracing

import (
	"context"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestExporterOptions(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		wantTarget string
		wantErr    bool
	}{
		{"nothing set: off", nil, "", false},
		{"standard OTLP endpoint wins", map[string]string{
			"OTEL_EXPORTER_OTLP_ENDPOINT": "http://phoenix:6006",
			"LANGFUSE_PUBLIC_KEY":         "pk", "LANGFUSE_SECRET_KEY": "sk-lf-secret",
		}, "http://phoenix:6006", false},
		{"traces endpoint before generic", map[string]string{
			"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://a/v1/traces",
			"OTEL_EXPORTER_OTLP_ENDPOINT":        "http://b",
		}, "http://a/v1/traces", false},
		{"langfuse cloud by default", map[string]string{"LANGFUSE_PUBLIC_KEY": "pk", "LANGFUSE_SECRET_KEY": "sk-lf-secret"},
			"https://cloud.langfuse.com/api/public/otel/v1/traces", false},
		{"langfuse custom host", map[string]string{
			"LANGFUSE_PUBLIC_KEY": "pk", "LANGFUSE_SECRET_KEY": "sk-lf-secret", "LANGFUSE_HOST": "https://us.cloud.langfuse.com/",
		}, "https://us.cloud.langfuse.com/api/public/otel/v1/traces", false},
		{"half a key pair is an error", map[string]string{"LANGFUSE_PUBLIC_KEY": "pk"}, "", true},
	}
	for _, tc := range cases {
		_, target, err := exporterOptions(envFrom(tc.env))
		if (err != nil) != tc.wantErr || target != tc.wantTarget {
			t.Errorf("%s: target=%q err=%v; want %q, error=%v", tc.name, target, err, tc.wantTarget, tc.wantErr)
		}
		if strings.Contains(target, "sk-lf-secret") {
			t.Errorf("%s: target leaks the secret key: %q", tc.name, target)
		}
	}
}

func TestSetup_NoEndpointIsNoop(t *testing.T) {
	for _, k := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "LANGFUSE_PUBLIC_KEY", "LANGFUSE_SECRET_KEY", "TRACE_LOG_CONTENT"} {
		t.Setenv(k, "")
	}
	shutdown, err := Setup(context.Background(), Config{Env: EnvEval})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	if Env() != EnvEval || !ContentEnabled() {
		t.Errorf("Env()=%q ContentEnabled()=%v", Env(), ContentEnabled())
	}

	t.Setenv("TRACE_LOG_CONTENT", "false")
	if _, err := Setup(context.Background(), Config{}); err != nil {
		t.Fatal(err)
	}
	if Env() != EnvLive || ContentEnabled() {
		t.Errorf("defaults: Env()=%q ContentEnabled()=%v; want live, false", Env(), ContentEnabled())
	}
}
