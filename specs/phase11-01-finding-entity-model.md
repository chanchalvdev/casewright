# Spec 11-01 — Finding and Entity model

## Title
One normalised `Finding` and one `Entity` vocabulary for every security domain.

## Description
Today the only thing the incident engine understands is a `models.ClassifiedEvent`
(a log line plus a verdict), and the only entities it can correlate on are `ip`,
`user` and `host`. A vulnerability scanner result, a cloud-posture check or an
identity anomaly has no way in. This spec introduces `finding.Finding` — the single
shape every connector produces and the incident engine consumes — and widens
`Entity` so findings from different domains that touch the same asset land in the
same incident. It is the keystone for every domain pack in Phase 11: one
correlation engine, one case workflow, one audit trail.

## Instructions

### Non-goals (keep v1 small)
- No full OCSF schema. Adopt OCSF *class names and field naming* where cheap, and
  pin one OCSF release in `docs/FINDINGS.md`; do not import the whole schema.
- No explicit entity-graph edges. Co-occurrence in a finding or incident *is* the
  edge. A real graph store is a later decision, driven by a query we cannot answer.
- No change to how detection rules or the LLM classifier work. They keep producing
  `ClassifiedEvent`; an adapter turns that into a `Finding`.

### Package `internal/finding` (new; every file ≤150 LOC)
- `finding.go`
  ```go
  type Domain string
  const (
      DomainDetection     Domain = "detection"     // log-based alerts (today's events)
      DomainVulnerability Domain = "vulnerability" // CVE / scanner findings
      DomainPosture       Domain = "posture"       // cloud / config misconfiguration
      DomainIdentity      Domain = "identity"      // sign-in / account anomalies
      DomainCompliance    Domain = "compliance"    // control failures
  )

  type Finding struct {
      ID         string            // deterministic, see below
      Domain     Domain
      Source     string            // connector name, e.g. "trivy", "syslog"
      Kind       string            // domain-specific, e.g. "sigma_match", "cve"
      Title      string
      Summary    string
      Severity   models.Severity   // reuse P1..P5; do not add a second scale
      Confidence float64           // 0..1; 0 means "not assessed"
      At         time.Time         // when the thing was observed (untrusted)
      Entities   []Entity
      MITRE      models.MITREInfo  // zero value when not applicable
      Attributes map[string]string // domain-specific, bounded (see Validate)
      Raw        string            // original payload, truncated (see Validate)
  }
  ```
- `Validate() error` rejects: empty `Source`/`Kind`/`Title`, unknown `Domain`,
  severity outside P1–P5, more than **32** entities, more than **32** attributes,
  attribute keys/values over **256** bytes, `Raw` over **16 KiB** (truncate, do not
  reject). Bounds exist because connectors, and therefore findings, are
  attacker-influenced input.
- `ID` is `sha256(domain|source|kind|primary entity|dedup fields)` hex, **not**
  random and **not** derived from `At`. The same CVE on the same image from the
  same scanner is the same finding on every scan; re-ingesting it updates, never
  duplicates. (Contrast with the event ID issue called out in
  `PLATFORM_EVOLUTION.md` §1.8.)
- `entity.go`
  ```go
  const (
      KindIP, KindUser, KindHost          = "ip", "user", "host"          // existing
      KindDomain, KindURL, KindFileHash   = "domain", "url", "file_hash"  // network / IOC
      KindRepo, KindImage, KindPackage    = "repo", "image", "package"    // supply chain
      KindCloudAccount, KindService       = "cloud_account", "service"
  )
  type Entity struct { Kind, Value string }
  func Canonical(e Entity) (Entity, error)
  func ParseEntity(s string) (Entity, error) // "repo:org/app", "ip:1.2.3.4"
  ```
  `Canonical` lower-cases hosts, domains and hashes, normalises IPs through
  `net/netip`, trims whitespace, and rejects empty values, control characters and
  values over 256 bytes. **Correlation correctness depends on this**: `Web01` and
  `web01` must be one host.
- `adapt.go` — `FromClassified(models.ClassifiedEvent) Finding`
  (`DomainDetection`, `Source` from `Event.Source`, `Kind` from `ClassifiedBy`)
  and `ToClassified(Finding) (models.ClassifiedEvent, bool)` for the reverse on
  detection findings, so existing code paths keep working during migration.

### Wiring into existing packages (smallest change that works)
1. `internal/incident`: make `Entity` a type alias, `type Entity = finding.Entity`,
   and delegate `ParseEntity` to `finding.ParseEntity`. JSON and DB representations
   are unchanged (`{"kind","value"}`), so **no data migration** is needed for
   existing incidents.
2. `incident.EntitiesOf(ev)` keeps its signature; internally it calls
   `finding.FromClassified` and returns `Finding.Entities`.
3. Add `incident.Service.ObserveFinding(ctx, f finding.Finding) (Result, bool, error)`.
   `Observe(ctx, ev)` becomes a one-line wrapper over it. Correlation, dedup,
   suppression and kill-chain logic are untouched.
4. Alerts store the originating `finding_id` and `domain` (additive JSON fields on
   `incident.Alert`; one migration `0005_findings.sql` adds nullable columns).
   Vulnerability findings have no MITRE tactic: the kill-chain view must skip
   alerts with an empty `Tactic` rather than invent one.
5. **Tenant seam:** every table this spec adds (`entities`, and the columns in
   `0005_findings.sql`) carries `tenant_id TEXT NOT NULL DEFAULT 'default'`, and
   unique keys include it, e.g. `(tenant_id, kind, value)`. Nothing reads it yet;
   multi-tenancy (Phase 9) is built last, and this keeps it from becoming a
   retrofit across every table.

### Ingestion
- `POST /api/findings` — accepts one `Finding` or `{"findings":[...]}` (max
  **100** per request). Auth: existing API key / session; role **analyst** or
  above. Each finding is `Validate`d, entities canonicalised, then passed through
  the same suppression → incident path as classified events. Response lists
  `accepted`, `duplicates` and per-item `errors`.
- Findings are **not** sent to the LLM classifier. They arrive with a verdict
  (severity, title) from their source; the investigation agent runs per incident,
  exactly as today.

### Asset context (v1: store and expose, do not act)
- Optional `Attributes` on an entity (criticality `low|medium|high|crit`, owner,
  environment, tags) live in a small `entities` table keyed by canonical
  `(kind, value)`, with `first_seen` / `last_seen` maintained on ingest.
- Exposed read-only: `GET /api/entities/{kind}/{value}` (finding + incident counts,
  attributes) and later as an agent tool (spec 11-02).
- **Deliberately not in v1:** automatic severity changes from criticality. Changing
  priority from a number an analyst typed into a form is a policy decision; do it
  as an explicit, testable rule in a later spec.

## Validation test
`internal/finding/*_test.go` and `internal/incident/*_test.go`:
1. **Canonical:** `Canonical({host, " Web01 "})` equals `{host, "web01"}`;
   `{ip, "2001:DB8::1"}` and `{ip, "2001:db8:0:0:0:0:0:1"}` canonicalise equal;
   a value containing `\x1b` or `\n` is rejected.
2. **Validate bounds:** 33 entities, an oversized attribute, an unknown domain and
   a severity of `"P9"` each return an error; an oversized `Raw` is truncated to
   16 KiB, not rejected.
3. **Deterministic ID:** same inputs produce the same ID; changing only `At` or
   `Summary` does not change it; changing `Source` does.
4. **Adapter round-trip:** `ToClassified(FromClassified(ev))` preserves severity,
   attack type, MITRE, IOCs and entities for a table of existing sample events.
5. **Backwards compatibility:** the entire existing `internal/incident` test suite
   passes unmodified after the alias change.
6. **Cross-domain correlation (the point of the spec):** observe a detection
   finding for `host:web01`, then a vulnerability finding for `host:web01` inside
   the correlation window; assert **one** incident containing both alerts.
7. **Idempotent ingest:** posting the same vulnerability finding twice yields
   `accepted:1` then `duplicates:1`, and `alert_count` does not increase.
8. **Kill-chain safety:** an incident containing a vulnerability finding with no
   tactic renders without error and omits that alert from the chain.
9. **Authorisation:** `viewer` receives 403 on `POST /api/findings`; no key
   receives 401.

## Acceptance criteria
- [ ] `internal/finding` exists, every file ≤150 LOC, has tests, `go vet` clean.
- [ ] `incident.Entity` is an alias; existing incident JSON and DB rows are
      byte-compatible; no breaking migration.
- [ ] A non-log finding can open and join incidents through `POST /api/findings`.
- [ ] Two findings from different domains sharing a canonical entity correlate
      into one incident (test 6).
- [ ] Re-ingesting an unchanged finding never creates a second alert (test 7).
- [ ] All bounds in `Validate` enforced; control characters cannot reach the
      database or the UI via entity values.
- [ ] `docs/FINDINGS.md` documents the schema, the pinned OCSF release, and how to
      write a connector that emits findings.
- [ ] `make quality-gate` passes; `go test -race ./...` green.
