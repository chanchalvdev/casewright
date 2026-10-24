# Using Casewright

Back to the [README](../README.md).

## Using the Dashboard

### Classify a log manually

Paste any log line in the input bar and press **Enter** or click **Classify**:

```
Failed password for root from 45.33.32.156 port 22
vssadmin.exe delete shadows /all /quiet
GET /etc/passwd HTTP/1.1 200
sudo: hacker USER=root COMMAND=/bin/bash
```

### Upload a log file

Click **Upload** and select any `.log` or `.txt` file. The UI classifies all lines in parallel batches and shows a progress bar.

A ready-made test file with 30 mixed-severity events is included:

```
casewright/test-logs.txt
```

### Severity filter

Use the sidebar checkboxes to filter events by P1–P5. Event counts per severity are shown live.

### Event detail

Click any event card to open the detail panel showing:
- Severity badge + confidence bar
- MITRE ATT&CK tactic + technique (links to attack.mitre.org)
- IOCs with VirusTotal / AbuseIPDB links
- Remediation steps (collapsible)
- Raw log
- Similar past events (semantic search)

### Analytics

Switch to the **Analytics** tab (mobile) or view the right panel (desktop) for:
- Critical / High event counters
- Attack type bar chart (color-coded by severity)
- Event rate timeline (6h, 10-min buckets)
- MITRE ATT&CK tactic pie chart
- MITRE heatmap and threat intel panel

### Live incidents

When the agent investigates a high-severity event, its progress streams in over
`/ws/alerts` and appears in the **alert ticker** at the top of the app. Click a
ticker entry to open the **incident overlay** — tool calls, findings, and the
final verdict update live as the agent works.

## CLI Mode

Classify a log file directly from the terminal without starting the HTTP server:

```bash
./bin/casewright sample.log
```

Output results to JSON:

```bash
./bin/casewright --output results.json sample.log
```

## Responsive Design

The dashboard is fully responsive:

| Breakpoint | Layout |
|---|---|
| Mobile (< 768px) | Hamburger sidebar, tab bar (Events / Analytics), bottom-sheet event detail |
| Tablet (768–1280px) | Collapsible sidebar, full event list, detail panel as overlay |
| Desktop (> 1280px) | Three-column layout — sidebar + events + detail/analytics panel |
