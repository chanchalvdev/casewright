import type { ClassifiedEvent } from '../lib/api'
import { SEVERITY_BORDER } from '../styles/tokens'
import { SeverityBadge } from './SeverityBadge'
import { MITREBadge } from './MITREBadge'
import type { Severity } from '../styles/tokens'

interface Props {
  event: ClassifiedEvent
  onClick: () => void
  selected: boolean
  score?: number
}

function timeAgo(iso: string): string {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  if (diff < 60)   return `${Math.round(diff)}s ago`
  if (diff < 3600) return `${Math.round(diff / 60)}m ago`
  if (diff < 86400) return `${Math.round(diff / 3600)}h ago`
  return `${Math.round(diff / 86400)}d ago`
}

export function EventCard({ event, onClick, selected, score }: Props) {
  const sev = event.severity as Severity
  const borderColor = SEVERITY_BORDER[sev]
  const selectedClass = selected
    ? 'ring-1 ring-blue-500 bg-gray-800/80'
    : 'bg-gray-900 hover:bg-gray-800/60'

  const summary = event.summary?.length > 80
    ? event.summary.slice(0, 80) + '…'
    : event.summary

  const confidence = Math.round((event.confidence ?? 0) * 100)

  return (
    <button
      onClick={onClick}
      className={`w-full text-left border-l-4 ${borderColor} ${selectedClass} rounded-r border border-l-4 border-gray-800 p-3 transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <SeverityBadge severity={sev} size="sm" />
          <span className="text-sm font-medium text-gray-200">{event.attack_type}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {score !== undefined && (
            <span className="text-xs text-blue-400 bg-blue-900/30 px-1.5 py-0.5 rounded">
              {score}% match
            </span>
          )}
          <span className="text-xs text-gray-500">
            {timeAgo(event.processed_at || event.event.timestamp)}
          </span>
        </div>
      </div>

      <div className="mt-1 flex items-center gap-2 text-xs text-gray-500">
        {event.event.hostname && <span>{event.event.hostname}</span>}
        {event.event.app_name && (
          <>
            <span>·</span>
            <span>{event.event.app_name}</span>
          </>
        )}
      </div>

      {summary && (
        <p className="mt-1.5 text-xs text-gray-400 font-log leading-relaxed">{summary}</p>
      )}

      <div className="mt-2 flex items-center justify-between gap-2">
        <MITREBadge
          tactic={event.mitre?.tactic ?? ''}
          technique={event.mitre?.technique_id ?? ''}
        />
        <div className="flex items-center gap-1.5 shrink-0">
          <div className="h-1.5 w-16 bg-gray-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full"
              style={{ width: `${confidence}%` }}
            />
          </div>
          <span className="text-xs text-gray-600">{confidence}%</span>
        </div>
      </div>
    </button>
  )
}
