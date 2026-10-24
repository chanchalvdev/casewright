// Package tracing exports OpenTelemetry spans over OTLP/HTTP. Any OTLP
// backend works; Langfuse (eval runs) and Arize Phoenix (live traffic, kept
// on-host) are the two this project uses.
package tracing

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Environments. Every span is tagged with one so eval runs never mix with
// real traffic in a dashboard.
const (
	EnvLive = "live"
	EnvEval = "eval"
)

const defaultLangfuseHost = "https://cloud.langfuse.com"

// Config describes this process to the tracing backend.
type Config struct {
	Env         string // EnvLive or EnvEval
	Version     string // build version
	LLMProvider string
}

var (
	env             atomic.Value // string
	contentDisabled atomic.Bool  // zero value: content on
)

// Env reports the environment spans are tagged with.
func Env() string {
	if v, ok := env.Load().(string); ok {
		return v
	}
	return EnvLive
}

// ContentEnabled reports whether spans may carry log lines and model answers.
// TRACE_LOG_CONTENT=false keeps only metadata (label, latency, tokens).
func ContentEnabled() bool { return !contentDisabled.Load() }

// Setup installs a global tracer provider when an endpoint is configured and
// returns a function that flushes pending spans. With no endpoint it installs
// nothing: spans stay no-ops and nothing leaves the process.
//
// Endpoint, in order: the standard OTEL_EXPORTER_OTLP_(TRACES_)ENDPOINT and
// OTEL_EXPORTER_OTLP_HEADERS variables; otherwise LANGFUSE_PUBLIC_KEY +
// LANGFUSE_SECRET_KEY (+ LANGFUSE_HOST, default Langfuse Cloud).
func Setup(ctx context.Context, c Config) (shutdown func(context.Context) error, err error) {
	if c.Env == "" {
		c.Env = EnvLive
	}
	env.Store(c.Env)
	contentDisabled.Store(strings.EqualFold(strings.TrimSpace(os.Getenv("TRACE_LOG_CONTENT")), "false"))

	noop := func(context.Context) error { return nil }
	opts, target, err := exporterOptions(os.Getenv)
	if err != nil || target == "" {
		return noop, err
	}

	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return noop, err
	}
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			attribute.String("service.name", "casewright"),
			attribute.String("service.version", c.Version),
			attribute.String("deployment.environment.name", c.Env),
			attribute.String("casewright.llm_provider", c.LLMProvider),
		),
	)
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)

	slog.Info("tracing enabled", "component", "tracing", "endpoint", target, "env", c.Env, "log_content", ContentEnabled())
	if c.Env == EnvLive && ContentEnabled() && strings.Contains(target, "langfuse.com") {
		slog.Warn("live traces include raw log lines and are going to a third-party cloud; use a self-hosted backend or TRACE_LOG_CONTENT=false",
			"component", "tracing")
	}
	return tp.Shutdown, nil
}

// exporterOptions resolves where spans go. It returns the target for logging
// (never the credentials) and an empty target when tracing is off.
func exporterOptions(getenv func(string) string) ([]otlptracehttp.Option, string, error) {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT"} {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return nil, v, nil // the exporter reads the OTEL_* variables itself
		}
	}

	pub, sec := strings.TrimSpace(getenv("LANGFUSE_PUBLIC_KEY")), strings.TrimSpace(getenv("LANGFUSE_SECRET_KEY"))
	if pub == "" && sec == "" {
		return nil, "", nil
	}
	if pub == "" || sec == "" {
		return nil, "", errors.New("set both LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY, or neither")
	}
	host := strings.TrimRight(strings.TrimSpace(getenv("LANGFUSE_HOST")), "/")
	if host == "" {
		host = defaultLangfuseHost
	}
	url := host + "/api/public/otel/v1/traces"
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(pub+":"+sec))
	return []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(url),
		otlptracehttp.WithHeaders(map[string]string{"Authorization": auth}),
	}, url, nil
}
