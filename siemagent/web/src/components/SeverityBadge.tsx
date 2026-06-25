import type { Severity } from '../styles/tokens'
import { SEVERITY_BG } from '../styles/tokens'

export const SEVERITY_LABELS: Record<Severity, string> = {
  P1: 'Critical',
  P2: 'High',
  P3: 'Medium',
  P4: 'Low',
  P5: 'Info',
}

interface Props {
  severity: Severity
  size?: 'sm' | 'md'
}

export function SeverityBadge({ severity, size = 'md' }: Props) {
  const sizeClass = size === 'sm'
    ? 'text-xs px-1.5 py-0.5'
    : 'text-sm px-2 py-1'

  const pulseClass = severity === 'P1' ? 'severity-pulse' : ''

  return (
    <span
      className={`inline-flex items-center font-mono font-semibold rounded border ${sizeClass} ${SEVERITY_BG[severity]} ${pulseClass}`}
    >
      {severity} · {SEVERITY_LABELS[severity]}
    </span>
  )
}
