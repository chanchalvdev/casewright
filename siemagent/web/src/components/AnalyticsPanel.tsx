import { useQuery } from '@tanstack/react-query'
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell,
  LineChart, Line, PieChart, Pie, Legend,
} from 'recharts'
import { getAnalytics } from '../lib/api'
import { SEVERITY_COLORS } from '../styles/tokens'

const CHART_BG = '#111827'      // gray-900
const AXIS_COLOR = '#4B5563'    // gray-600
const TEXT_COLOR = '#9CA3AF'    // gray-400

export function AnalyticsPanel() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['analytics'],
    queryFn: getAnalytics,
    refetchInterval: 30_000,
    staleTime: 15_000,
  })

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        {[220, 180, 160].map(h => (
          <div key={h} style={{ height: h }} className="bg-gray-800 rounded animate-pulse" />
        ))}
      </div>
    )
  }

  if (isError || !data) {
    return (
      <p className="p-4 text-sm text-gray-600">Analytics unavailable.</p>
    )
  }

  // ── Attack-type bar chart data ──────────────────────────────────────────────
  const attackData = Object.entries(data.attack_type_counts)
    .sort((a, b) => b[1].count - a[1].count)
    .slice(0, 12)
    .map(([name, info]) => ({
      name: name.length > 18 ? name.slice(0, 16) + '…' : name,
      count: info.count,
      severity: info.severity,
    }))

  // ── Timeline line chart data ────────────────────────────────────────────────
  const timelineData = data.timeline.map(b => ({
    time: b.time,
    P1: b.counts['P1'] ?? 0,
    P2: b.counts['P2'] ?? 0,
    P3: b.counts['P3'] ?? 0,
  }))

  // ── MITRE tactic pie chart data ─────────────────────────────────────────────
  const tacticData = Object.entries(data.mitre_tactics)
    .filter(([t]) => t && t !== 'N/A')
    .sort((a, b) => b[1] - a[1])
    .slice(0, 8)
    .map(([name, value]) => ({ name, value }))

  const PIE_COLORS = [
    '#3B82F6', '#8B5CF6', '#EC4899', '#EF4444',
    '#F97316', '#EAB308', '#10B981', '#06B6D4',
  ]

  return (
    <div className="space-y-6 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300">Analytics</h2>
        <span className="text-xs text-gray-600">{data.total_events} events in memory</span>
      </div>

      {/* Attack-type bar chart */}
      <div>
        <p className="text-xs text-gray-500 mb-2">Events by attack type (last 1000)</p>
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={attackData} margin={{ left: -10 }}>
            <XAxis
              dataKey="name"
              tick={{ fill: TEXT_COLOR, fontSize: 10 }}
              axisLine={{ stroke: AXIS_COLOR }}
              tickLine={false}
              interval={0}
              angle={-30}
              textAnchor="end"
              height={50}
            />
            <YAxis tick={{ fill: TEXT_COLOR, fontSize: 10 }} axisLine={false} tickLine={false} />
            <Tooltip
              contentStyle={{ background: CHART_BG, border: '1px solid #374151', color: TEXT_COLOR, fontSize: 12 }}
              cursor={{ fill: 'rgba(255,255,255,0.05)' }}
            />
            <Bar dataKey="count" radius={[3, 3, 0, 0]}>
              {attackData.map((entry, i) => (
                <Cell key={i} fill={SEVERITY_COLORS[entry.severity as keyof typeof SEVERITY_COLORS] ?? '#6B7280'} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      {/* Timeline line chart */}
      <div>
        <p className="text-xs text-gray-500 mb-2">Event rate over time (last 6h, 10-min buckets)</p>
        <ResponsiveContainer width="100%" height={160}>
          <LineChart data={timelineData} margin={{ left: -10 }}>
            <XAxis
              dataKey="time"
              tick={{ fill: TEXT_COLOR, fontSize: 10 }}
              axisLine={{ stroke: AXIS_COLOR }}
              tickLine={false}
              interval={5}
            />
            <YAxis tick={{ fill: TEXT_COLOR, fontSize: 10 }} axisLine={false} tickLine={false} />
            <Tooltip
              contentStyle={{ background: CHART_BG, border: '1px solid #374151', color: TEXT_COLOR, fontSize: 12 }}
            />
            <Line type="monotone" dataKey="P1" stroke={SEVERITY_COLORS.P1} dot={false} strokeWidth={2} />
            <Line type="monotone" dataKey="P2" stroke={SEVERITY_COLORS.P2} dot={false} strokeWidth={2} />
            <Line type="monotone" dataKey="P3" stroke={SEVERITY_COLORS.P3} dot={false} strokeWidth={2} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* MITRE tactic pie chart */}
      {tacticData.length > 0 && (
        <div>
          <p className="text-xs text-gray-500 mb-2">MITRE ATT&CK tactic distribution</p>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={tacticData}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius={70}
                label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                labelLine={false}
              >
                {tacticData.map((_, i) => (
                  <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{ background: CHART_BG, border: '1px solid #374151', color: TEXT_COLOR, fontSize: 12 }}
              />
              <Legend wrapperStyle={{ color: TEXT_COLOR, fontSize: 11 }} />
            </PieChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}
