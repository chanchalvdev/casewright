package store

import (
	"sync"
	"time"

	"github.com/chverma/siemagent/internal/models"
)

const maxEvents = 1000

// EventStore is a thread-safe ring buffer of the last N classified events.
type EventStore struct {
	mu     sync.RWMutex
	events [maxEvents]models.ClassifiedEvent
	head   int // next write position
	count  int // total stored (capped at maxEvents)
}

func New() *EventStore { return &EventStore{} }

// Add appends a classified event to the ring buffer.
func (s *EventStore) Add(ev models.ClassifiedEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[s.head] = ev
	s.head = (s.head + 1) % maxEvents
	if s.count < maxEvents {
		s.count++
	}
}

// Recent returns the most recent n events (newest first).
func (s *EventStore) Recent(n int) []models.ClassifiedEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n > s.count {
		n = s.count
	}
	out := make([]models.ClassifiedEvent, n)
	for i := range n {
		idx := (s.head - 1 - i + maxEvents) % maxEvents
		out[i] = s.events[idx]
	}
	return out
}

// ─── Analytics ────────────────────────────────────────────────────────────────

type AttackCount struct {
	Count    int    `json:"count"`
	Severity string `json:"severity"` // highest severity seen for this type
}

type TimelineBucket struct {
	Time  string         `json:"time"`
	Counts map[string]int `json:"counts"` // severity -> count
}

type AnalyticsSummary struct {
	TotalEvents      int                    `json:"total_events"`
	AttackTypeCounts map[string]*AttackCount `json:"attack_type_counts"`
	Timeline         []TimelineBucket        `json:"timeline"` // 10-min buckets, last 6h
	MITRETactics     map[string]int          `json:"mitre_tactics"`
}

var severityOrder = map[models.Severity]int{
	models.SeverityP1: 1,
	models.SeverityP2: 2,
	models.SeverityP3: 3,
	models.SeverityP4: 4,
	models.SeverityP5: 5,
}

func higherSeverity(a, b string) string {
	oa := severityOrder[models.Severity(a)]
	ob := severityOrder[models.Severity(b)]
	if oa < ob {
		return a
	}
	return b
}

// Summary computes analytics over the buffered events.
func (s *EventStore) Summary() AnalyticsSummary {
	events := s.Recent(maxEvents)

	now := time.Now().UTC()
	cutoff := now.Add(-6 * time.Hour)

	// Build 10-minute bucket labels for the last 6h (36 buckets).
	numBuckets := 36
	buckets := make([]TimelineBucket, numBuckets)
	for i := range numBuckets {
		t := cutoff.Add(time.Duration(i) * 10 * time.Minute)
		buckets[i] = TimelineBucket{
			Time:  t.Format("15:04"),
			Counts: make(map[string]int),
		}
	}

	attackCounts := make(map[string]*AttackCount)
	mitreTactics := make(map[string]int)

	for _, ev := range events {
		// Attack type counts
		ac, ok := attackCounts[ev.AttackType]
		if !ok {
			ac = &AttackCount{Severity: string(ev.Severity)}
			attackCounts[ev.AttackType] = ac
		}
		ac.Count++
		ac.Severity = higherSeverity(ac.Severity, string(ev.Severity))

		// MITRE tactic distribution
		if ev.MITRE.Tactic != "" && ev.MITRE.Tactic != "N/A" {
			mitreTactics[ev.MITRE.Tactic]++
		}

		// Timeline bucket
		if ev.ProcessedAt.After(cutoff) {
			offset := int(ev.ProcessedAt.Sub(cutoff).Minutes() / 10)
			if offset >= 0 && offset < numBuckets {
				sev := string(ev.Severity)
				if sev == "P1" || sev == "P2" || sev == "P3" {
					buckets[offset].Counts[sev]++
				}
			}
		}
	}

	return AnalyticsSummary{
		TotalEvents:      len(events),
		AttackTypeCounts: attackCounts,
		Timeline:         buckets,
		MITRETactics:     mitreTactics,
	}
}
