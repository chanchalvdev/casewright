# Platform Evolution — Critical Review & Expansion Beyond SIEM

> A deliberately critical companion to [CASE_STUDY.md](CASE_STUDY.md) and [specs/README.md](specs/README.md).
> Those documents describe what was built and that it works. This one argues about
> what is missing, what is structurally weak, and what this engine could become if
> it stops thinking of itself as "a SIEM agent."
>
> Written to be disagreed with. Every claim cites a file so it can be checked.

---

## Part 0 — The one-paragraph verdict

The project is a **well-engineered demonstration of an idea**, not yet a product.
The Go is clean, the tests pass, the agent loop is properly bounded, the tool
registry is a genuinely good abstraction. But the system **has no memory, no
evaluation, and no cheap tier**: it forgets everything on restart, it cannot tell
you whether a prompt change made classification better or worse, and it sends
100% of events to an LLM. Those three gaps — not the absence of features — are
what separate this from something deployable. Adding more verticals before fixing
them would be building a second floor on an unfinished foundation.

---

## Part 1 — Critical review of what actually exists

### 1.1 The architecture is economically impossible at real volume

Every event takes the same path: HTTP → parse → **LLM** → response
([`classifier.go:105`](siemagent/internal/classifier/classifier.go#L105)). There is
no rule tier, no cache, no dedup, no fingerprinting — confirmed absent across the
whole module.

A real SOC ingests millions of events per day, and the overwhelming majority are
identical, boring, and repetitive: the same cron job, the same health check, the
same successful login. Sending each of those to an LLM with a ~700-token system
prompt is the architectural equivalent of hiring a senior analyst to read every
line of `/var/log/syslog` by hand.

The stated throughput target in CLAUDE.md — **>50 events/min** — quietly admits
this. Fifty events per minute is 72,000/day. A single mid-sized web server
produces more than that before lunch.

**What a real system does:** a deterministic tier (Sigma rules, regex, allow-lists,
dedup by fingerprint) absorbs 95–99% of volume at microsecond cost, and the LLM is
an *escalation path* for the ambiguous remainder. That inversion is the single
highest-value change available to this codebase, and it makes every other feature
cheaper.

### 1.2 The system has amnesia

[`store/events.go:10-18`](siemagent/internal/store/events.go#L10-L18) is a
**1000-element in-memory ring buffer**. Restart the process and every classified
event, every analytics number, every incident is gone. The dashboard's analytics
(`Summary()`) are computed over whatever happens to still be in RAM.

Meanwhile `docker-compose.yml` provisions **Postgres, which no Go code references**
(grep for `postgres|pgx|database/sql` returns nothing outside compose). The
infrastructure was staged and never wired.

A security tool that cannot answer *"what happened last Tuesday?"* is not a SIEM.
Retention and queryable history are not a nice-to-have in this domain; they are
frequently the actual legal requirement (PCI-DSS wants a year).

### 1.3 There is no way to know if the classifier is any good

No golden dataset. No regression scoring. No labelled fixtures. No accuracy metric
of any kind — grep for `eval|golden|feedback` returns nothing.

This is the most serious gap for an *LLM* product specifically. Right now, if
someone edits the system prompt at
[`classifier.go:18`](siemagent/internal/classifier/classifier.go#L18) — reorders a
severity rule, adds an attack type — **there is no mechanism on earth to tell
whether the product got better or worse.** The Go tests verify that JSON parses,
not that the verdict is correct. `go test` passing means the plumbing works, not
that the intelligence works.

Every prompt change is therefore an unmeasured gamble. That is the difference
between an LLM demo and an LLM product, and it is worth more than any feature on
the roadmap.

### 1.4 The most valuable data asset is being thrown away

When an analyst looks at a verdict and thinks "that's a false positive" — nothing
captures it. There is no feedback endpoint, no verdict override, no label store.

This is a strategic loss, not just a missing feature. Qdrant is *already wired*
([`classifier.go:160`](siemagent/internal/classifier/classifier.go#L160)) and holds
embeddings of past events. Confirmed analyst verdicts could be retrieved as
few-shot examples on every future classification, making the system measurably
better at *this customer's* logs over time. That compounding loop is the only part
of this product that a competitor cannot copy in a weekend. It is currently not
built.

### 1.5 Events are judged in isolation — attacks are not

Every classification sees exactly one log line
([`buildUserMessage`](siemagent/internal/classifier/classifier.go#L198)). But an
intrusion is a *sequence*:

```
failed login ×50  →  successful login  →  sudo su  →  outbound 40GB
   (P3, boring)      (P5, "normal")     (P4)        (P1, too late)
```

Classified independently, the first three are individually unremarkable and the
alert fires only at the exfiltration — the exact failure mode that lets breaches
run for months. **Correlation across time, host, and user is the core analytical
capability of a SIEM, and it does not exist here.** The vector store makes this
unusually tractable, which makes the omission more frustrating, not less.

### 1.6 Cost is completely unmeasured

Three Prometheus metrics exist (`events_classified_total`,
`classification_duration_seconds`, `llm_stream_errors_total`) and **none of them
track tokens or money**. No `TokensUsed`, no `PromptTokens`, no cost accounting
anywhere in the module.

The product's dominant operating expense is invisible to its own observability
stack. You cannot answer "what does it cost to classify 10,000 events?" — which is
the first question any buyer asks, and the first question that determines whether
the architecture in §1.1 is viable.

### 1.7 Security posture of the security product

- **No inbound authentication on any route.** The `APIKey` in config is the
  *outbound* LLM credential. `/api/classify`, `/api/search`, `/api/analytics` are
  all open ([`server.go:124-129`](siemagent/internal/api/server.go#L124-L129)).
- **`/metrics` is publicly served** ([`server.go:121`](siemagent/internal/api/server.go#L121)) —
  it leaks attack types and volumes to anyone who asks.
- **The agent has no spend ceiling.** `maxIterations = 8`
  ([`agent.go:16`](siemagent/internal/agent/agent.go#L16)) bounds the loop count but
  not tokens, not wall-clock, not concurrent investigations. Auto-triggering on
  high-severity events means an attacker who can generate P1-looking logs can
  generate unbounded LLM spend — a cost-exhaustion vector.
- **Credit where due — two defences are properly built.** IOC validation before
  outbound requests rejects malformed IPs *and* private/loopback/link-local ranges
  ([`abuseipdb.go:56-60`](siemagent/internal/agent/tools/abuseipdb.go#L56-L60),
  [`otx.go:63-67`](siemagent/internal/agent/tools/otx.go#L63-L67)) — a correct SSRF
  defence on the highest-risk path, where LLM-extracted strings become outbound
  requests. Separately, [`sanitize.go`](siemagent/internal/api/sanitize.go) strips
  ANSI escapes and C0 control characters from log-derived output, defending the
  *viewer* against log-injection. Both are real, and both are the kind of thing
  that usually gets skipped.

### 1.8 Smaller correctness and discipline issues

| Issue | Location | Note |
|---|---|---|
| Hand-rolled `min()` | [`classifier.go:191`](siemagent/internal/classifier/classifier.go#L191) | CLAUDE.md explicitly mandates the Go 1.21+ builtin |
| `fmt.Fprintf(os.Stderr,…)` in prod path | [`pool.go:41`](siemagent/internal/pipeline/pool.go#L41) | CLAUDE.md mandates `slog` — worker errors are invisible to structured logging |
| Event ID is a truncated raw string | [`classifier.go:175`](siemagent/internal/classifier/classifier.go#L175) | `Raw[:32]` as identity — collides on similar prefixes |
| Identical logs create distinct vectors | [`classifier.go:185`](siemagent/internal/classifier/classifier.go#L185) | ID includes `UnixNano`, so the same line re-indexes forever; Qdrant grows without bound |
| Five packages ship no tests | `config`, `metrics`, `models`, `pipeline`, `store` | CLAUDE.md: "All new Go packages must have at least one `_test.go`" — `pipeline` and `store` hold real logic |
| Silent worker drop | [`pool.go:41-43`](siemagent/internal/pipeline/pool.go#L41-L43) | Classify failure `continue`s — the event is lost with no DLQ, no retry, no counter |
| Harness is fiction | `scripts/`, `.github/workflows/`, root `Makefile` | CLAUDE.md documents a quality-gate harness and CI that **do not exist as files** |

That last row deserves emphasis: CLAUDE.md describes `make quality-gate`,
`self-improve.sh`, and two GitHub Actions workflows in confident detail. None are
real. Documentation that describes non-existent infrastructure is worse than no
documentation, because it is trusted.

---

## Part 2 — The reframe: what this engine actually is

Strip the security vocabulary out of the codebase and look at the shape of it:

```
ingest text  →  normalize to a struct  →  LLM produces a typed verdict
   →  map verdict onto a domain ontology  →  embed into vector memory
   →  agentic tool-use enrichment  →  stream the result live to a UI
```

**Not one word of that pipeline is inherently about security.** "SIEM" is the
current *skin*: the system prompt, the MITRE ATT&CK ontology, and four threat-intel
tools. The spine underneath is a general **semantic event-intelligence platform**.

That is why "why keep this only a SIEM agent?" is the right question. The honest
answer is that SIEM is one vertical on a horizontal engine — and the parts that
make it a SIEM are, by design, the *most swappable* parts of the codebase:

| Layer | Security-specific? | Swap cost |
|---|---|---|
| Parser (`internal/parser`) | Partly (syslog) | Low — add a format |
| **System prompt** (`classifier.go:18`) | **Entirely** | **Trivial — it's a string** |
| **Ontology** (MITRE mapping) | **Entirely** | **Low — `MITREInfo` → a generic taxonomy struct** |
| **Tools** (`agent/tools/*`) | **Entirely** | **Low — the `Registry` already abstracts this** |
| Severity scale (P1–P5) | No | None — universal |
| Vector memory, agent loop, WS hub, pool, API | No | None — fully reusable |

The tool registry in particular was built well enough that adding a tool is a
contained, single-file change. That abstraction is the project's most valuable
asset and it is currently being used for exactly one domain.

---

## Part 3 — Adjacent products the same spine supports

Ordered by *leverage*: how much of the existing codebase survives untouched.

### 3.1 AppSec finding triage — highest leverage, real pain

Security scanners (SAST, dependency, container, cloud) emit thousands of findings
of which most are unexploitable in context. Teams drown; the tooling has a
notorious signal-to-noise problem.

- **Input swap:** scanner JSON instead of syslog.
- **Ontology swap:** MITRE ATT&CK → CWE/CVSS + reachability.
- **Tool swap:** threat-intel → EPSS score, KEV catalogue, "is this code path
  actually reachable?", "is this dependency in the runtime bundle?"
- **Reused unchanged:** agent loop, dedup/clustering via vectors, severity scale,
  streaming UI, incident playbook generation.

This is the strongest candidate because LLMs are genuinely good at the core
judgement ("is this finding exploitable *here*?"), the pain is universally
acknowledged, and the vector store solves finding-deduplication almost for free.

### 3.2 Detection engineering copilot — closes the loop with §1.1

Generate, test, and maintain Sigma/YARA detection rules *from observed events*.
The system watches what the LLM classifies as malicious and proposes deterministic
rules to catch the same thing next time — for free.

This is elegant because it directly solves the cost problem in §1.1: the LLM stops
being the runtime detector and becomes the thing that *writes the runtime
detectors*. LLM cost becomes a one-time authoring cost instead of a per-event tax.
**If only one idea in this document gets built, it should probably be this one.**

### 3.3 Detection coverage & log quality analytics

Answer: *"Given the log sources we actually ingest, which ATT&CK techniques could
we even theoretically detect?"* Most organisations cannot answer this and it
terrifies them.

Near-zero new infrastructure — it is analytics over data already being classified,
plus a coverage matrix UI (the `MITREHeatmap` component already exists). Very high
perceived value per unit of engineering effort.

### 3.4 Compliance evidence automation

Every classification is already a timestamped, structured audit artifact. Map it
to SOC 2 / ISO 27001 / PCI-DSS controls instead of ATT&CK techniques and the same
pipeline generates continuous compliance evidence — a budget line that is far
easier to get signed than "better alerts."

Requires §1.2 (durable storage) as a hard prerequisite; evidence you cannot retain
is not evidence.

### 3.5 SRE / observability incident triage

Same engine, different noun. Application logs, traces, and alerts instead of
security events; root-cause hypothesis instead of attack classification; service
dependency graph instead of ATT&CK; tools query Prometheus/Loki/K8s instead of
AbuseIPDB.

Large market, but note the honest catch: this is a crowded field with
well-funded incumbents, and it is a genuinely *different buyer* from security.
High technical leverage, low commercial leverage for a solo project.

### 3.6 Insider threat / UEBA

Per-entity behavioural baselines — "this user has never touched this file share at
3am." The vector store is the natural substrate for "is this normal *for this
entity*?" Requires §1.2 and §1.5 first; without history and correlation there is
no baseline to deviate from.

### 3.7 Others worth naming but not pursuing yet

- **Cloud posture (CSPM/CIEM)** — CloudTrail/K8s audit input, CIS benchmark ontology.
- **Fraud & abuse detection** — payment/account events; identical shape, different
  regulatory surface.
- **Standalone threat-intel enrichment API** — expose `agent/tools` as its own
  product; small but genuinely useful.
- **Post-incident report generation** — already ~80% delivered by the existing
  playbook generator; mostly packaging.

---

## Part 4 — Platform capabilities that pay off in every vertical

These are horizontal: build once, every vertical above inherits them.

| # | Capability | Why it matters | Depends on |
|---|---|---|---|
| P1 | **Eval harness + golden dataset** | Makes every future change measurable instead of a gamble (§1.3) | — |
| P2 | **Deterministic pre-filter tier** | 10–100× cost reduction; makes real volume viable (§1.1) | — |
| P3 | **Durable storage + audit trail** | Postgres is already provisioned and unused (§1.2) | — |
| P4 | **Analyst feedback loop** | The only compounding, non-copyable advantage (§1.4) | P1, P3 |
| P5 | **Correlation engine** | Turns event classification into incident detection (§1.5) | P3 |
| P6 | **Cost/token accounting** | Makes the dominant opex visible (§1.6) | — |
| P7 | **Auth, tenancy, spend caps** | Prerequisite for anyone else touching it (§1.7) | P3 |
| P8 | **Model routing & fallback** | Cheap model for easy events, strong model for hard ones; survive provider outage | P1, P6 |
| P9 | **Air-gapped / local-only mode** | Ollama is already wired — "your logs never leave your network" is a genuine differentiator in security, where data residency kills deals | — |
| P10 | **Connector framework** | Kafka, syslog listener, Filebeat, webhooks — pluggable ingestion | P2 |

**P9 deserves more attention than it usually gets.** The single loudest objection
to LLM-based security tooling is "we are not shipping our logs to a third-party
model." This codebase already talks to an OpenAI-compatible endpoint and already
runs Ollama locally. A fully local, air-gapped deployment mode is *mostly
configuration and proof*, and it neutralises the objection that blocks the most
deals. Low effort, disproportionate strategic value.

---

## Part 5 — What I would actually do, in order

### Stage 1 — Earn the right to add features (nothing new ships until these do)

1. **P1 eval harness.** 100–200 hand-labelled log events, a scoring run
   (accuracy, severity MAE, IOC recall, MITRE agreement), a committed baseline.
   Every prompt change reports a delta from then on.
2. **P3 persistence.** Wire the Postgres that compose already starts. Durable
   events, durable incidents, retention policy, real analytics over real history.
3. **P6 cost metrics.** Tokens in/out per classification, per-event cost, agent
   spend per investigation.
4. **Fix §1.8.** Builtin `min`, `slog` in the worker pool, a DLQ + counter instead
   of a silent `continue`, tests for `store` and `pipeline`.
5. **Make CLAUDE.md true.** Either build the harness it describes or delete the
   claim. Documented-but-absent infrastructure is a trust bug.

### Stage 2 — Build the things that are hard to copy

6. **P2 rules tier.** Sigma-compatible deterministic matching in front of the LLM,
   with fingerprint dedup and a verdict cache. Measure the cost delta with P6 and
   prove the 10×.
7. **P4 feedback loop.** Verdict override endpoint, label store, retrieval of
   confirmed labels as few-shot context. Prove it moved the P1 numbers.
8. **P5 correlation.** Session/entity windowing, sequence-aware incident grouping.
   This is where it stops being a classifier and starts being a SIEM.

### Stage 3 — Expand, but only once

9. **Pick exactly one vertical.** My recommendation: **3.2 detection engineering
   copilot**, because it reuses the most, directly reinforces Stage 2's rules tier,
   and is a story no one else is telling well. **3.1 AppSec triage** is the
   strongest commercial alternative.
10. **P9 air-gapped mode** alongside it — cheap, and it unlocks the buyer who was
    never going to say yes otherwise.

### What to explicitly not do yet

- **Do not build SOAR auto-remediation.** Auto-blocking IPs from a
  non-deterministic classifier with no audit trail and no eval harness is a
  liability, not a feature. It needs P1, P3, and P5 first — and a human approval
  gate permanently.
- **Do not build multi-tenancy before a second tenant exists.** It is a large,
  invasive change justified by a hypothetical.
- **Do not chase three verticals at once.** Each one needs credibility with a
  different audience. Three shallow verticals lose to one that is actually good.
- **Do not add another LLM feature before P1 exists.** Every one added now is
  unmeasurable, and they compound into a system nobody can safely change.

---

## Part 6 — The honest strategic summary

The instinct behind "why keep this only a SIEM agent?" is correct: the engine is
genuinely general, and the security-specific parts are the most swappable code in
the repository. There is a real platform here.

But the honest read is that **breadth is not this project's current bottleneck —
depth is.** A second vertical built on a foundation that forgets everything on
restart, cannot measure its own accuracy, and sends every event to an LLM does not
produce two products. It produces the same unfinished product twice, and doubles
the surface area that the missing eval harness fails to cover.

The sequencing that respects both instincts: **make it real (Stage 1), make it
defensible (Stage 2), then make it broad (Stage 3).** The expansion ideas in
Part 3 do not expire. The opportunity to build the feedback loop before
accumulating unlabelled data does.

One last framing worth sitting with: of everything in this document, the only item
a competitor genuinely cannot replicate quickly is **P4, the analyst feedback
loop** — a system that gets measurably better at *your* logs the longer it runs.
Every other capability here, including every vertical in Part 3, is a matter of
someone else's engineering time. That one is a matter of accumulated data, and it
only starts compounding on the day it is built.
