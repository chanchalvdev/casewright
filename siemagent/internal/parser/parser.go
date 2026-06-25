package parser

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/chverma/siemagent/internal/models"
)

// rfc5424RE matches RFC 5424 syslog lines.
// <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID [SD] MSG
var rfc5424RE = regexp.MustCompile(
	`^<(\d{1,3})>(\d)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(?:\[.*?\]\s*)?(.*)$`,
)

// rfc3164RE matches the older BSD syslog (RFC 3164) format.
// <PRI>TIMESTAMP HOSTNAME TAG: MSG
var rfc3164RE = regexp.MustCompile(
	`^<(\d{1,3})>(\w{3}\s+\d+\s+[\d:]+)\s+(\S+)\s+(\S+?):\s+(.*)$`,
)

type Parser struct{}

func New() *Parser { return &Parser{} }

// ParseLine attempts RFC 5424, RFC 3164, JSON, then falls back to raw.
func (p *Parser) ParseLine(line string) []models.LogEvent {
	return p.ParseLineWithFormat(line, "auto")
}

// ParseLineWithFormat parses a line respecting the given format hint.
// format must be "syslog", "json", or "auto".
func (p *Parser) ParseLineWithFormat(line, format string) []models.LogEvent {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	switch format {
	case "syslog":
		if ev, ok := p.parseRFC5424(line); ok {
			return []models.LogEvent{ev}
		}
		if ev, ok := p.parseRFC3164(line); ok {
			return []models.LogEvent{ev}
		}
		return []models.LogEvent{p.ParseRaw(line)}
	case "json":
		if ev, ok := p.parseJSON(line); ok {
			return []models.LogEvent{ev}
		}
		return []models.LogEvent{p.ParseRaw(line)}
	default: // "auto"
		if ev, ok := p.parseRFC5424(line); ok {
			return []models.LogEvent{ev}
		}
		if ev, ok := p.parseRFC3164(line); ok {
			return []models.LogEvent{ev}
		}
		if ev, ok := p.parseJSON(line); ok {
			return []models.LogEvent{ev}
		}
		return []models.LogEvent{p.ParseRaw(line)}
	}
}

func (p *Parser) parseRFC5424(line string) (models.LogEvent, bool) {
	m := rfc5424RE.FindStringSubmatch(line)
	if m == nil {
		return models.LogEvent{}, false
	}
	ts, _ := time.Parse(time.RFC3339Nano, m[3])
	if ts.IsZero() {
		ts, _ = time.Parse(time.RFC3339, m[3])
	}
	hostname := nilDash(m[4])
	appName := nilDash(m[5])
	procID := nilDash(m[6])
	msg := strings.TrimSpace(m[8])
	msg = strings.TrimPrefix(msg, "\xef\xbb\xbf")
	return models.LogEvent{
		Raw:       line,
		Timestamp: ts,
		Hostname:  hostname,
		AppName:   appName,
		ProcID:    procID,
		Message:   msg,
		Source:    "syslog",
	}, true
}

func (p *Parser) parseRFC3164(line string) (models.LogEvent, bool) {
	m := rfc3164RE.FindStringSubmatch(line)
	if m == nil {
		return models.LogEvent{}, false
	}
	ts, _ := time.Parse("Jan  2 15:04:05", m[2])
	if ts.IsZero() {
		ts, _ = time.Parse("Jan _2 15:04:05", m[2])
	}
	if !ts.IsZero() {
		ts = time.Date(time.Now().Year(), ts.Month(), ts.Day(),
			ts.Hour(), ts.Minute(), ts.Second(), 0, time.UTC)
	}
	return models.LogEvent{
		Raw:       line,
		Timestamp: ts,
		Hostname:  m[3],
		AppName:   m[4],
		Message:   strings.TrimSpace(m[5]),
		Source:    "syslog",
	}, true
}

func (p *Parser) parseJSON(line string) (models.LogEvent, bool) {
	if !strings.HasPrefix(line, "{") {
		return models.LogEvent{}, false
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return models.LogEvent{}, false
	}

	ev := models.LogEvent{Raw: line, Source: "json"}

	if v, ok := extractString(raw, "timestamp", "@timestamp", "time", "ts"); ok {
		t, _ := time.Parse(time.RFC3339Nano, v)
		if t.IsZero() {
			t, _ = time.Parse(time.RFC3339, v)
		}
		ev.Timestamp = t
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}

	ev.Hostname, _ = extractString(raw, "hostname", "host", "source_host")
	ev.AppName, _ = extractString(raw, "app", "application", "program", "service", "syslog_identifier")
	ev.ProcID, _ = extractString(raw, "pid", "proc_id", "process_id")

	if msg, ok := extractString(raw, "message", "msg", "log", "MESSAGE"); ok {
		ev.Message = msg
	} else {
		ev.Message = line
	}
	return ev, true
}

// ParseRaw creates a raw log event with no parsing.
func (p *Parser) ParseRaw(line string) models.LogEvent {
	return models.LogEvent{
		Raw:       line,
		Timestamp: time.Now().UTC(),
		Message:   line,
		Source:    "raw",
	}
}

func nilDash(s string) string {
	if s == "-" || s == "" {
		return ""
	}
	return s
}

func extractString(m map[string]interface{}, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}
