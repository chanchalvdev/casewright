import { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search as SearchIcon, AlertCircle } from 'lucide-react'
import { searchEvents } from '../lib/api'
import type { Severity } from '../styles/tokens'
import { SeverityBadge, SEVERITY_LABELS } from '../components/SeverityBadge'
import { MITREBadge } from '../components/MITREBadge'

const ALL_SEVERITIES: Severity[] = ['P1', 'P2', 'P3', 'P4', 'P5']

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useState(() => {
    const t = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(t)
  })
  return debounced
}

export function SearchPage() {
  const [query, setQuery] = useState('')
  const [severityFilter, setSeverityFilter] = useState<Severity | ''>('')
  const debouncedQuery = useDebounce(query, 300)

  const { data: results = [], isFetching, isError } = useQuery({
    queryKey: ['search', debouncedQuery, severityFilter],
    queryFn: () => searchEvents(debouncedQuery, severityFilter || undefined, 20),
    enabled: debouncedQuery.length > 2,
    staleTime: 30_000,
  })

  const handleSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault()
  }, [])

  function timeAgo(iso: string) {
    const diff = (Date.now() - new Date(iso).getTime()) / 1000
    if (diff < 60) return `${Math.round(diff)}s ago`
    if (diff < 3600) return `${Math.round(diff / 60)}m ago`
    return `${Math.round(diff / 3600)}h ago`
  }

  return (
    <div className="flex flex-col h-screen bg-gray-950 text-gray-100">
      {/* Header */}
      <header className="sticky top-0 z-10 bg-gray-950/90 backdrop-blur border-b border-gray-800 px-6 py-4">
        <form onSubmit={handleSubmit} className="flex gap-3 max-w-3xl">
          <div className="relative flex-1">
            <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
            <input
              autoFocus
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Search events by description, host, attack type…"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg pl-9 pr-4 py-2.5 text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-600"
            />
          </div>
          <select
            value={severityFilter}
            onChange={e => setSeverityFilter(e.target.value as Severity | '')}
            className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2.5 text-sm text-gray-300 focus:outline-none focus:border-blue-600"
          >
            <option value="">All severities</option>
            {ALL_SEVERITIES.map(s => (
              <option key={s} value={s}>{s} · {SEVERITY_LABELS[s]}</option>
            ))}
          </select>
        </form>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-4">
        {/* States */}
        {query.length <= 2 && (
          <div className="flex flex-col items-center justify-center h-full text-gray-600 text-center">
            <SearchIcon size={48} className="mb-4 opacity-20" />
            <p className="text-sm">Type at least 3 characters to search</p>
            <p className="text-xs mt-1">Searches by semantic similarity across all indexed events</p>
          </div>
        )}

        {query.length > 2 && isFetching && (
          <div className="flex items-center justify-center h-32 text-gray-500 text-sm">
            Searching…
          </div>
        )}

        {isError && (
          <div className="flex items-center gap-2 text-red-400 bg-red-900/20 border border-red-800 rounded px-4 py-3 text-sm">
            <AlertCircle size={16} />
            Search unavailable — Qdrant may not be running.
          </div>
        )}

        {!isFetching && query.length > 2 && results.length === 0 && !isError && (
          <div className="flex flex-col items-center justify-center h-64 text-gray-600 text-center">
            <SearchIcon size={48} className="mb-4 opacity-20" />
            <p className="text-sm">No results for "{query}"</p>
          </div>
        )}

        {/* Results */}
        <div className="space-y-2 max-w-3xl">
          {results.map((hit, i) => (
            <div
              key={`${hit.event_id}-${i}`}
              className="bg-gray-900 border border-gray-800 rounded-lg p-4 hover:border-gray-700 transition-colors"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-2 flex-wrap">
                  <SeverityBadge severity={hit.severity} size="sm" />
                  <span className="text-sm font-medium text-gray-200">{hit.attack_type}</span>
                  <span className="text-xs text-blue-400 bg-blue-900/30 px-1.5 py-0.5 rounded font-mono">
                    {Math.round(hit.score * 100)}% match
                  </span>
                </div>
                <span className="text-xs text-gray-500 shrink-0">{timeAgo(hit.timestamp)}</span>
              </div>

              {hit.summary && (
                <p className="mt-2 text-sm text-gray-400 font-log leading-relaxed">{hit.summary}</p>
              )}

              <div className="mt-2 flex items-center gap-2">
                <span className="text-xs text-gray-600">{hit.source}</span>
                {hit.mitre_tactic && hit.mitre_tactic !== 'N/A' && (
                  <MITREBadge tactic={hit.mitre_tactic} technique="" />
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
