import { useState, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Shield, Search, AlertTriangle, RefreshCw } from 'lucide-react'
import { classifyLog } from '../lib/api'
import type { ClassifiedEvent } from '../lib/api'
import type { Severity } from '../styles/tokens'
import { EventCard } from '../components/EventCard'
import { SeverityBadge } from '../components/SeverityBadge'
import { MITREBadge } from '../components/MITREBadge'
import { IOCList } from '../components/IOCList'
import { DropZone } from '../components/DropZone'
import { SimilarEvents } from '../components/SimilarEvents'
import { AnalyticsPanel } from '../components/AnalyticsPanel'

const ALL_SEVERITIES: Severity[] = ['P1', 'P2', 'P3', 'P4', 'P5']

const EVENTS_KEY = 'classified-events'

function useEventStore() {
  const qc = useQueryClient()

  const { data: events = [] } = useQuery<ClassifiedEvent[]>({
    queryKey: [EVENTS_KEY],
    queryFn: () => [],
    staleTime: Infinity,
  })

  function addEvent(ev: ClassifiedEvent) {
    qc.setQueryData<ClassifiedEvent[]>([EVENTS_KEY], (prev = []) => [ev, ...prev])
  }

  function addEvents(evs: ClassifiedEvent[]) {
    qc.setQueryData<ClassifiedEvent[]>([EVENTS_KEY], (prev = []) => [...evs.reverse(), ...prev])
  }

  function clear() {
    qc.setQueryData<ClassifiedEvent[]>([EVENTS_KEY], [])
  }

  return { events, addEvent, addEvents, clear }
}

export function Dashboard() {
  const { events, addEvent, addEvents, clear } = useEventStore()
  const [selected, setSelected] = useState<ClassifiedEvent | null>(null)
  const [logInput, setLogInput] = useState('')
  const [classifying, setClassifying] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [severityFilter, setSeverityFilter] = useState<Set<Severity>>(new Set(ALL_SEVERITIES))
  const inputRef = useRef<HTMLInputElement>(null)

  const filtered = events.filter((e) => severityFilter.has(e.severity as Severity))

  async function handleClassify(e: React.FormEvent) {
    e.preventDefault()
    if (!logInput.trim() || classifying) return
    setError(null)
    setClassifying(true)
    try {
      const ev = await classifyLog(logInput.trim())
      addEvent(ev)
      setSelected(ev)
      setLogInput('')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    } finally {
      setClassifying(false)
    }
  }

  function toggleSeverity(s: Severity) {
    setSeverityFilter((prev) => {
      const next = new Set(prev)
      if (next.has(s)) next.delete(s)
      else next.add(s)
      return next
    })
  }

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100 overflow-hidden">
      {/* Sidebar */}
      <aside className="w-60 shrink-0 border-r border-gray-800 flex flex-col">
        <div className="px-4 py-5 border-b border-gray-800">
          <div className="flex items-center gap-2">
            <Shield size={20} className="text-blue-400" />
            <span className="font-semibold text-white">SIEMAgent</span>
          </div>
          <p className="text-xs text-gray-500 mt-0.5">AI Security Classifier</p>
        </div>

        <nav className="px-2 py-4 space-y-1">
          <div className="px-2 py-1.5 rounded bg-gray-800 text-sm text-white flex items-center gap-2">
            <AlertTriangle size={15} className="text-blue-400" />
            Dashboard
          </div>
          <div className="px-2 py-1.5 rounded text-sm text-gray-400 hover:text-gray-200 flex items-center gap-2 cursor-pointer">
            <Search size={15} />
            Search
          </div>
        </nav>

        <div className="px-4 py-4 border-t border-gray-800 mt-auto">
          <p className="text-xs text-gray-500 uppercase tracking-wider mb-2">Severity Filter</p>
          <div className="space-y-1.5">
            {ALL_SEVERITIES.map((s) => (
              <label key={s} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={severityFilter.has(s)}
                  onChange={() => toggleSeverity(s)}
                  className="rounded border-gray-600 bg-gray-800 accent-blue-500"
                />
                <SeverityBadge severity={s} size="sm" />
              </label>
            ))}
          </div>
          {events.length > 0 && (
            <button
              onClick={clear}
              className="mt-3 w-full text-xs text-gray-500 hover:text-gray-300 flex items-center gap-1"
            >
              <RefreshCw size={11} /> Clear all ({events.length})
            </button>
          )}
        </div>
      </aside>

      {/* Main */}
      <div className="flex flex-1 min-w-0">
        <div className="flex flex-col flex-1 min-w-0">
          {/* Header */}
          <header className="sticky top-0 z-10 bg-gray-950/90 backdrop-blur border-b border-gray-800 px-4 py-3 flex items-center gap-3">
            <DropZone onResults={addEvents} />
            <form onSubmit={handleClassify} className="flex-1 flex gap-2">
              <input
                ref={inputRef}
                value={logInput}
                onChange={(e) => setLogInput(e.target.value)}
                placeholder="Paste a log line and press Enter to classify…"
                className="flex-1 bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm font-log text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-600"
              />
              <button
                type="submit"
                disabled={classifying || !logInput.trim()}
                className="px-3 py-1.5 rounded bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium transition-colors"
              >
                {classifying ? 'Classifying…' : 'Classify'}
              </button>
            </form>
          </header>

          {error && (
            <div className="mx-4 mt-3 text-sm text-red-400 bg-red-900/20 border border-red-800 rounded px-3 py-2">
              {error}
            </div>
          )}

          {/* Event list */}
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
            {filtered.length === 0 && (
              <div className="flex flex-col items-center justify-center h-full text-center text-gray-600 py-20">
                <Shield size={48} className="mb-4 opacity-20" />
                <p className="text-sm">No events yet.</p>
                <p className="text-xs mt-1">Paste a log line above or upload a file to get started.</p>
              </div>
            )}
            {filtered.map((ev, i) => (
              <EventCard
                key={`${ev.processed_at}-${i}`}
                event={ev}
                onClick={() => setSelected(ev === selected ? null : ev)}
                selected={ev === selected}
              />
            ))}
          </div>
        </div>

        {/* Detail panel */}
        {selected && (
          <aside className="w-96 shrink-0 border-l border-gray-800 overflow-y-auto">
            <div className="p-4 space-y-4">
              <div className="flex items-start justify-between gap-2">
                <SeverityBadge severity={selected.severity as Severity} size="md" />
                <button
                  onClick={() => setSelected(null)}
                  className="text-gray-500 hover:text-gray-300 text-lg leading-none"
                >
                  ×
                </button>
              </div>

              <div>
                <h2 className="text-base font-semibold text-white">{selected.attack_type}</h2>
                <p className="text-sm text-gray-400 mt-1">{selected.summary}</p>
              </div>

              {selected.mitre && (
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider mb-1.5">MITRE ATT&CK</p>
                  <MITREBadge
                    tactic={selected.mitre.tactic}
                    technique={selected.mitre.technique_id}
                  />
                  {selected.mitre.technique && (
                    <p className="text-xs text-gray-500 mt-1">{selected.mitre.technique}</p>
                  )}
                </div>
              )}

              {selected.iocs?.length > 0 && (
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider mb-1.5">Indicators of Compromise</p>
                  <IOCList iocs={selected.iocs} />
                </div>
              )}

              {selected.remediation && (
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider mb-1.5">Recommended Action</p>
                  <p className="text-sm text-gray-300 leading-relaxed">{selected.remediation}</p>
                </div>
              )}

              <div>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-1.5">Confidence</p>
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-2 bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-blue-500 rounded-full"
                      style={{ width: `${Math.round((selected.confidence ?? 0) * 100)}%` }}
                    />
                  </div>
                  <span className="text-sm text-gray-400">{Math.round((selected.confidence ?? 0) * 100)}%</span>
                </div>
              </div>

              <div>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-1.5">Raw Log</p>
                <pre className="text-xs font-log text-gray-400 bg-gray-900 rounded p-3 overflow-x-auto whitespace-pre-wrap break-all border border-gray-800">
                  {selected.event.raw}
                </pre>
              </div>

              <div className="text-xs text-gray-600">
                <span>Source: {selected.event.source}</span>
                {selected.event.hostname && <span> · {selected.event.hostname}</span>}
                {selected.event.app_name && <span> / {selected.event.app_name}</span>}
              </div>

              {/* Similar past events via RAG */}
              {selected.summary && (
                <div className="border-t border-gray-800 pt-4">
                  <SimilarEvents summary={selected.summary} />
                </div>
              )}
            </div>
          </aside>
        )}

        {/* Analytics panel — shown when no event is selected */}
        {!selected && (
          <aside className="w-80 shrink-0 border-l border-gray-800 overflow-y-auto">
            <AnalyticsPanel />
          </aside>
        )}
      </div>
    </div>
  )
}
