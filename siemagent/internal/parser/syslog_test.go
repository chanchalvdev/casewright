package parser

import (
	"testing"
	"time"
)

func TestParseSyslog_RFC5424(t *testing.T) {
	p := New()

	tests := []struct {
		name     string
		line     string
		wantOk   bool // should produce source=="syslog"
		hostname string
		appName  string
		procID   string
		message  string
		utc      bool // timestamp should be UTC
	}{
		{
			name:     "valid RFC5424",
			line:     `<165>1 2024-01-15T10:30:00Z webserver sshd 1234 - Failed password for root from 192.168.1.100 port 22`,
			wantOk:   true,
			hostname: "webserver",
			appName:  "sshd",
			procID:   "1234",
			message:  "Failed password for root from 192.168.1.100 port 22",
			utc:      true,
		},
		{
			name:     "NILVALUE procid dash",
			line:     `<34>1 2024-01-15T10:30:00Z host app - - some message`,
			wantOk:   true,
			hostname: "host",
			appName:  "app",
			procID:   "", // "-" should become empty
			message:  "some message",
		},
		{
			name:     "UTC timestamp preserved",
			line:     `<13>1 2024-06-01T00:00:00Z myhost myapp 42 - hello`,
			wantOk:   true,
			utc:      true,
			hostname: "myhost",
			appName:  "myapp",
			procID:   "42",
			message:  "hello",
		},
		{
			name:   "empty string returns nil",
			line:   "",
			wantOk: false,
		},
		{
			name:   "plain text without PRI header falls back to raw",
			line:   "Jun 15 10:30:00 myhost sshd: Failed password for root",
			wantOk: false, // not RFC5424, may be 3164 or raw
		},
		{
			// RFC 5424: [UFW BLOCK] is the structured-data element; the parser
			// strips it per the grammar, so the message starts after the SD.
			name:     "kernel log with PID 0 and brackets in message",
			line:     `<0>1 2024-01-15T10:30:00Z localhost kernel 0 - [UFW BLOCK] IN=eth0 OUT= MAC=00:11:22 SRC=10.0.0.1 DST=192.168.1.1 PROTO=TCP`,
			wantOk:   true,
			hostname: "localhost",
			appName:  "kernel",
			procID:   "0",
			message:  "IN=eth0 OUT= MAC=00:11:22 SRC=10.0.0.1 DST=192.168.1.1 PROTO=TCP",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			events := p.ParseLine(tc.line)

			if tc.line == "" {
				if len(events) != 0 {
					t.Errorf("expected nil events for empty line, got %d", len(events))
				}
				return
			}

			if len(events) == 0 {
				if tc.wantOk {
					t.Fatal("expected at least one event, got none")
				}
				return
			}

			ev := events[0]

			if tc.wantOk && ev.Source != "syslog" {
				t.Errorf("source: want syslog, got %q", ev.Source)
			}
			if !tc.wantOk && ev.Source == "syslog" {
				t.Errorf("source: expected non-syslog fallback, got syslog")
			}

			if tc.wantOk {
				if tc.hostname != "" && ev.Hostname != tc.hostname {
					t.Errorf("hostname: want %q, got %q", tc.hostname, ev.Hostname)
				}
				if tc.appName != "" && ev.AppName != tc.appName {
					t.Errorf("appName: want %q, got %q", tc.appName, ev.AppName)
				}
				if tc.procID != "" && ev.ProcID != tc.procID {
					t.Errorf("procID: want %q, got %q", tc.procID, ev.ProcID)
				}
				if tc.message != "" && ev.Message != tc.message {
					t.Errorf("message: want %q, got %q", tc.message, ev.Message)
				}
				if tc.utc && !ev.Timestamp.IsZero() {
					if ev.Timestamp.Location() != time.UTC && ev.Timestamp.Location().String() != "UTC" {
						t.Errorf("expected UTC timestamp, got zone %q", ev.Timestamp.Location())
					}
				}
				// Only check NILVALUE when the test explicitly expects empty procID.
				if tc.procID == "" && ev.ProcID != "" {
					t.Errorf("NILVALUE procID: expected empty, got %q", ev.ProcID)
				}
			}
		})
	}
}
