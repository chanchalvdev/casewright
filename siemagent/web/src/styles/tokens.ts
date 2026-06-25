export type Severity = 'P1' | 'P2' | 'P3' | 'P4' | 'P5'

export const SEVERITY_COLORS: Record<Severity, string> = {
  P1: '#DC2626', // red-600
  P2: '#EA580C', // orange-600
  P3: '#CA8A04', // yellow-600
  P4: '#2563EB', // blue-600
  P5: '#6B7280', // gray-500
}

export const SEVERITY_BG: Record<Severity, string> = {
  P1: 'bg-red-900/30 text-red-400 border-red-700',
  P2: 'bg-orange-900/30 text-orange-400 border-orange-700',
  P3: 'bg-yellow-900/30 text-yellow-400 border-yellow-700',
  P4: 'bg-blue-900/30 text-blue-400 border-blue-700',
  P5: 'bg-gray-800 text-gray-400 border-gray-700',
}

export const SEVERITY_BORDER: Record<Severity, string> = {
  P1: 'border-l-red-600',
  P2: 'border-l-orange-600',
  P3: 'border-l-yellow-600',
  P4: 'border-l-blue-600',
  P5: 'border-l-gray-500',
}

export function useDarkMode() {
  // App is always dark-first; this hook is a stub for future toggle support
  return { isDark: true }
}
