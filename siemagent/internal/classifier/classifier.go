package classifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/trace"

	"github.com/chverma/siemagent/internal/metrics"
	"github.com/chverma/siemagent/internal/models"
)

var jsonBlockRE = regexp.MustCompile(`(?s)\{.*\}`)

// ErrInvalidOutput marks an LLM answer that is not decodable JSON, as opposed
// to a transport failure such as a timeout.
var ErrInvalidOutput = errors.New("invalid LLM output")

// Interface allows mocking in tests.
type Interface interface {
	Classify(ctx context.Context, ev models.LogEvent) (models.ClassifiedEvent, error)
	ClassifyStream(ctx context.Context, ev models.LogEvent, onChunk func(string)) (models.ClassifiedEvent, error)
	Ping(ctx context.Context) error
}

// Indexer is implemented by the Qdrant store; kept as an interface so tests
// that do not wire Qdrant still compile and pass.
type Indexer interface {
	Upsert(ctx context.Context, id string, vector []float32, payload map[string]any) error
}

// VectorEmbedder is implemented by the Ollama client.
type VectorEmbedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Classifier struct {
	client        *openai.Client
	model         string
	prompt        string // system prompt; empty = DefaultPromptVersion
	promptVersion string
	seed          *int           // optional; set by evals for repeatable runs
	streamUsage   bool           // ask for token counts in the stream (provider must support it)
	embedder      VectorEmbedder // optional; nil = no indexing
	indexer       Indexer        // optional; nil = no indexing
}

func New(apiKey, baseURL, model string) *Classifier {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return &Classifier{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// UsePrompt switches to an embedded prompt version, failing on unknown ones.
func (c *Classifier) UsePrompt(version string) error {
	p, err := Prompt(version)
	if err != nil {
		return err
	}
	c.prompt, c.promptVersion = p, version
	return nil
}

// PromptVersion reports which prompt version produces results.
func (c *Classifier) PromptVersion() string {
	if c.prompt == "" {
		return DefaultPromptVersion
	}
	return c.promptVersion
}

// SetSeed asks the provider for repeatable sampling (supported by Ollama and
// OpenAI). Used by evals; live classification leaves it unset.
func (c *Classifier) SetSeed(seed int) { c.seed = &seed }

// SetStreamUsage asks the provider to report token counts at the end of each
// stream. Only enable it for providers known to accept stream_options.
func (c *Classifier) SetStreamUsage(on bool) { c.streamUsage = on }

func (c *Classifier) systemPrompt() string {
	if c.prompt == "" {
		p, err := Prompt(DefaultPromptVersion)
		if err != nil {
			panic(err) // the default prompt is embedded; a test guards it
		}
		return p
	}
	return c.prompt
}

// WithIndexing wires optional Qdrant embedding+indexing into the classifier.
func (c *Classifier) WithIndexing(embedder VectorEmbedder, indexer Indexer) {
	c.embedder = embedder
	c.indexer = indexer
}

// OpenAIClient exposes the underlying LLM client so the agent can reuse
// the same configured connection.
func (c *Classifier) OpenAIClient() *openai.Client { return c.client }

// Ping sends a minimal request to verify the LLM is reachable.
func (c *Classifier) Ping(ctx context.Context) error {
	_, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "ping"},
		},
		// max_tokens, not max_completion_tokens: the OpenAI-compatible APIs of
		// Ollama and Gemini accept it, and a rejected Ping fails /health/ready.
		MaxTokens: 1, //nolint:staticcheck // deprecated in go-openai for o1 models only
	})
	return err
}

// Classify classifies a log event using the LLM, recording Prometheus metrics.
func (c *Classifier) Classify(ctx context.Context, ev models.LogEvent) (models.ClassifiedEvent, error) {
	return c.ClassifyStream(ctx, ev, nil)
}

// ClassifyStream streams the LLM response, calling onChunk for each text delta.
// If onChunk is nil it is ignored. Each call is one trace span.
func (c *Classifier) ClassifyStream(ctx context.Context, ev models.LogEvent, onChunk func(string)) (models.ClassifiedEvent, error) {
	ctx, span := tracer.Start(ctx, "chat "+c.model, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	var call llmCall
	call.input = buildUserMessage(ev)
	result, err := c.classify(ctx, ev, &call, onChunk)
	c.recordSpan(span, call, result, err)
	return result, err
}

// llmCall is what one request sent and received, kept for the trace span.
type llmCall struct {
	input  string
	output string
	usage  *openai.Usage
}

func (c *Classifier) classify(ctx context.Context, ev models.LogEvent, call *llmCall, onChunk func(string)) (models.ClassifiedEvent, error) {
	start := time.Now()
	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: c.systemPrompt()},
			{Role: openai.ChatMessageRoleUser, Content: call.input},
		},
		Temperature: 0.1,
		Seed:        c.seed,
	}
	if c.streamUsage {
		req.StreamOptions = &openai.StreamOptions{IncludeUsage: true}
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		metrics.LLMStreamErrorsTotal.Inc()
		return models.ClassifiedEvent{}, fmt.Errorf("stream open: %w", err)
	}
	defer func() { _ = stream.Close() }()

	var sb strings.Builder
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// A broken stream is a transport failure, not a bad answer:
			// report it as such instead of failing to parse a partial reply.
			metrics.LLMStreamErrorsTotal.Inc()
			call.output = sb.String()
			return models.ClassifiedEvent{}, fmt.Errorf("stream recv: %w", err)
		}
		if chunk.Usage != nil {
			call.usage = chunk.Usage
		}
		if len(chunk.Choices) > 0 {
			text := chunk.Choices[0].Delta.Content
			if text != "" {
				if onChunk != nil {
					onChunk(text)
				}
				sb.WriteString(text)
			}
		}
	}

	elapsed := time.Since(start).Seconds()
	metrics.ClassificationDurationSeconds.Observe(elapsed)
	if u := call.usage; u != nil {
		metrics.LLMTokensTotal.WithLabelValues("input").Add(float64(u.PromptTokens))
		metrics.LLMTokensTotal.WithLabelValues("output").Add(float64(u.CompletionTokens))
	}

	raw := sb.String()
	call.output = raw
	analysis, err := decodeLLM(raw)
	if err != nil {
		metrics.LLMInvalidOutputTotal.WithLabelValues(IssueParseError).Inc()
		return models.ClassifiedEvent{}, fmt.Errorf("parse LLM response: %w: %w (raw: %.200s)", ErrInvalidOutput, err, raw)
	}
	// Validate before normalising: live traffic still gets a usable verdict,
	// but the broken contract is counted and reported, never hidden.
	issues := validate(analysis)
	for _, issue := range issues {
		metrics.LLMInvalidOutputTotal.WithLabelValues(issue).Inc()
	}

	result := buildClassifiedEvent(ev, normalise(analysis))
	result.PromptVersion = c.PromptVersion()
	result.OutputIssues = issues
	metrics.EventsClassifiedTotal.WithLabelValues(string(result.Severity), metricLabel(result.AttackType)).Inc()

	// Asynchronously embed and index into Qdrant — does not block the response.
	if c.embedder != nil && c.indexer != nil {
		go c.indexEvent(result)
	}

	return result, nil
}

// Index stores an event produced elsewhere (e.g. by detection rules) for
// semantic search, asynchronously. No-op when indexing is not configured.
func (c *Classifier) Index(ev models.ClassifiedEvent) {
	if c.embedder != nil && c.indexer != nil {
		go c.indexEvent(ev)
	}
}

func (c *Classifier) indexEvent(ev models.ClassifiedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	text := fmt.Sprintf("%s %s %s %s %s",
		ev.Event.Source, ev.Event.AppName, ev.Event.Message,
		ev.AttackType, ev.MITRE.Tactic)

	vec, err := c.embedder.Embed(ctx, text)
	if err != nil {
		slog.Warn("embed failed", "component", "classifier", "error", err)
		return
	}

	payload := map[string]any{
		"event_id":     ev.Event.Raw[:min(32, len(ev.Event.Raw))],
		"timestamp":    ev.ProcessedAt.Format(time.RFC3339),
		"source":       ev.Event.Source,
		"hostname":     ev.Event.Hostname,
		"attack_type":  ev.AttackType,
		"severity":     string(ev.Severity),
		"summary":      ev.Summary,
		"mitre_tactic": ev.MITRE.Tactic,
	}

	id := fmt.Sprintf("%s-%d", ev.Event.Raw[:min(16, len(ev.Event.Raw))], ev.ProcessedAt.UnixNano())
	if err := c.indexer.Upsert(ctx, id, vec, payload); err != nil {
		slog.Warn("qdrant upsert failed", "component", "classifier", "error", err)
	}
}

func buildUserMessage(ev models.LogEvent) string {
	var sb strings.Builder
	sb.WriteString("Analyze this security log event:\n\n")
	if ev.Source != "raw" {
		fmt.Fprintf(&sb, "Source: %s\n", ev.Source)
		if !ev.Timestamp.IsZero() {
			fmt.Fprintf(&sb, "Timestamp: %s\n", ev.Timestamp.Format(time.RFC3339))
		}
		if ev.Hostname != "" {
			fmt.Fprintf(&sb, "Hostname: %s\n", ev.Hostname)
		}
		if ev.AppName != "" {
			fmt.Fprintf(&sb, "Application: %s\n", ev.AppName)
		}
		if ev.ProcID != "" {
			fmt.Fprintf(&sb, "PID: %s\n", ev.ProcID)
		}
		fmt.Fprintf(&sb, "Message: %s\n", ev.Message)
		sb.WriteString("\nRaw log line:\n")
	}
	sb.WriteString(ev.Raw)
	return sb.String()
}

// unmarshalLLM decodes and normalises an LLM answer.
func unmarshalLLM(raw string) (models.LLMAnalysis, error) {
	a, err := decodeLLM(raw)
	if err != nil {
		return models.LLMAnalysis{}, err
	}
	return normalise(a), nil
}

// decodeLLM extracts the JSON object from an LLM answer without changing any
// field, so validate sees exactly what the model returned.
func decodeLLM(raw string) (models.LLMAnalysis, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code fences
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	if !strings.HasPrefix(raw, "{") {
		if m := jsonBlockRE.FindString(raw); m != "" {
			raw = m
		}
	}

	var a models.LLMAnalysis
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		return models.LLMAnalysis{}, err
	}
	return a, nil
}

// normalise makes an answer safe to use: allowed labels get their canonical
// spelling, and an unknown severity becomes P5.
func normalise(a models.LLMAnalysis) models.LLMAnalysis {
	if l, ok := ParseLabel(a.AttackType); ok {
		a.AttackType = string(l)
	}
	a.Severity = strings.ToUpper(strings.TrimSpace(a.Severity))
	switch a.Severity {
	case "P1", "P2", "P3", "P4", "P5":
	default:
		a.Severity = "P5"
	}

	if a.IOCs == nil {
		a.IOCs = []string{}
	}
	return a
}

func buildClassifiedEvent(ev models.LogEvent, a models.LLMAnalysis) models.ClassifiedEvent {
	return models.ClassifiedEvent{
		Event:      ev,
		AttackType: a.AttackType,
		MITRE: models.MITREInfo{
			Tactic:      a.Tactic,
			TechniqueID: a.TechniqueID,
			Technique:   a.Technique,
		},
		Severity:     models.Severity(a.Severity),
		ClassifiedBy: models.ClassifiedByLLM,
		Confidence:   a.Confidence,
		IOCs:         a.IOCs,
		Remediation:  a.Remediation,
		Summary:      a.Summary,
		ProcessedAt:  time.Now().UTC(),
	}
}
