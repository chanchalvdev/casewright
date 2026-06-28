export type Severity = 'P1' | 'P2' | 'P3' | 'P4' | 'P5'

export const SEVERITY_COLORS: Record<Severity, string> = {
  P1: '#DC2626', // red-600
  P2: '#EA580C', // orange-600
  P3: '#CA8A04', // yellow-600
  P4: '#2563EB', // blue-600
  P5: '#6B7280', // gray-500
}

export const SEVERITY_LABELS: Record<Severity, string> = {
  P1: 'Critical',
  P2: 'High',
  P3: 'Medium',
  P4: 'Low',
  P5: 'Info',
}

export const SEVERITY_BG: Record<Severity, string> = {
  P1: 'bg-red-500/15 text-red-400 border-red-500/30',
  P2: 'bg-orange-500/15 text-orange-400 border-orange-500/30',
  P3: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
  P4: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  P5: 'bg-gray-500/15 text-gray-400 border-gray-500/30',
}

export const SEVERITY_BORDER: Record<Severity, string> = {
  P1: 'border-l-red-500',
  P2: 'border-l-orange-500',
  P3: 'border-l-yellow-500',
  P4: 'border-l-blue-500',
  P5: 'border-l-gray-500',
}

export function useDarkMode() {
  // App is always dark-first; this hook is a stub for future toggle support
  return { isDark: true }
}
