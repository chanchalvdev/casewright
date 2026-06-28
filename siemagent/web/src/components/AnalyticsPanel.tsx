import { useQuery } from '@tanstack/react-query'
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell,
  LineChart, Line, PieChart, Pie, Legend,
} from 'recharts'
import { getAnalytics } from '../lib/api'
import { SEVERITY_COLORS } from '../styles/tokens'
import { TrendingUp, Shield, Activity } from 'lucide-react'

const AXIS_COLOR  = '#374151'
const TEXT_COLOR  = '#6B7280'
const PIE_COLORS  = ['#3B82F6','#8B5CF6','#EC4899','#EF4444','#F97316','#EAB308','#10B981','#06B6D4']

const TOOLTIP_STYLE = {
  background: '#0d1426',
  border: '1px solid rgba(255,255,255,0.08)',
  borderRadius: 8,
  color: '#D1D5DB',
  fontSize: 11,
  padding: '6px 10px',
}

export function AnalyticsPanel() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['analytics'],
    queryFn: getAnalytics,
    refetchInterval: 30_000,
    staleTime: 15_000,
  })

  if (isLoading) {
    return (
      <div className="p-4 space-y-4">
        {[180, 140, 180].map((h, i) => (
          <div key={i} className="rounded-xl bg-white/4 animate-pulse" style={{ height: h }} />
        ))}
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="flex flex-col items-center justify-center h-40 text-center px-4">
        <Activity size={24} className="text-gray-700 mb-2" />
        <p className="text-xs text-gray-600">Analytics unavailable</p>
      </div>
    )
  }

  const attackData = Object.entries(data.attack_type_counts)
    .sort((a, b) => b[1].count - a[1].count)
    .slice(0, 10)
    .map(([name, info]) => ({
      name: name.length > 16 ? name.slice(0, 14) + '…' : name,
      count: info.count,
      severity: info.severity,
    }))

  const timelineData = data.timeline.map(b => ({
    time: b.time,
    P1: b.counts['P1'] ?? 0,
    P2: b.counts['P2'] ?? 0,
    P3: b.counts['P3'] ?? 0,
  }))

  const tacticData = Object.entries(data.mitre_tactics)
    .filter(([t]) => t && t !== 'N/A')
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6)
    .map(([name, value]) => ({ name, value }))

  const criticalCount = Object.values(data.attack_type_counts)
    .filter(v => v.severity === 'P1').reduce((a, v) => a + v.count, 0)
  const highCount = Object.values(data.attack_type_counts)
    .filter(v => v.severity === 'P2').reduce((a, v) => a + v.count, 0)

  return (
    <div className="p-4 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-200 flex items-center gap-2">
          <TrendingUp size={15} className="text-blue-400" />
          Analytics
        </h2>
        <span className="text-[10px] text-gray-600">{data.total_events} events total</span>
      </div>

      {/* Quick stat cards */}
      <div className="grid grid-cols-2 gap-2">
        <div className="bg-red-950/30 border border-red-900/30 rounded-xl p-3">
          <p className="text-[10px] text-red-400/70 uppercase tracking-widest">Critical</p>
          <p className="text-2xl font-bold text-red-400 mt-0.5">{criticalCount}</p>
        </div>
        <div className="bg-orange-950/30 border border-orange-900/30 rounded-xl p-3">
          <p className="text-[10px] text-orange-400/70 uppercase tracking-widest">High</p>
          <p className="text-2xl font-bold text-orange-400 mt-0.5">{highCount}</p>
        </div>
      </div>

      {/* Attack type bar chart */}
      {attackData.length > 0 && (
        <div className="bg-white/3 border border-white/5 rounded-xl p-3">
          <p className="text-[10px] text-gray-500 uppercase tracking-widest mb-3 flex items-center gap-1.5">
            <Shield size={11} /> Attack Types
          </p>
          <ResponsiveContainer width="100%" height={180}>
            <BarChart data={attackData} margin={{ left: -20, right: 4, top: 4, bottom: 36 }}>
              <XAxis
                dataKey="name"
                tick={{ fill: TEXT_COLOR, fontSize: 9 }}
                axisLine={{ stroke: AXIS_COLOR }}
                tickLine={false}
                interval={0}
                angle={-35}
                textAnchor="end"
                height={50}
              />
              <YAxis
                tick={{ fill: TEXT_COLOR, fontSize: 9 }}
                axisLine={false}
                tickLine={false}
                allowDecimals={false}
              />
              <Tooltip contentStyle={TOOLTIP_STYLE} cursor={{ fill: 'rgba(255,255,255,0.03)' }} />
              <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                {attackData.map((entry, i) => (
                  <Cell key={i} fill={SEVERITY_COLORS[entry.severity as keyof typeof SEVERITY_COLORS] ?? '#374151'} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Timeline */}
      <div className="bg-white/3 border border-white/5 rounded-xl p-3">
        <p className="text-[10px] text-gray-500 uppercase tracking-widest mb-3 flex items-center gap-1.5">
          <Activity size={11} /> Event Rate (6h)
        </p>
        <ResponsiveContainer width="100%" height={120}>
          <LineChart data={timelineData} margin={{ left: -20, right: 4, top: 4, bottom: 0 }}>
            <XAxis
              dataKey="time"
              tick={{ fill: TEXT_COLOR, fontSize: 9 }}
              axisLine={{ stroke: AXIS_COLOR }}
              tickLine={false}
              interval={5}
            />
            <YAxis
              tick={{ fill: TEXT_COLOR, fontSize: 9 }}
              axisLine={false}
              tickLine={false}
              allowDecimals={false}
            />
            <Tooltip contentStyle={TOOLTIP_STYLE} />
            <Line type="monotone" dataKey="P1" stroke={SEVERITY_COLORS.P1} dot={false} strokeWidth={2} />
            <Line type="monotone" dataKey="P2" stroke={SEVERITY_COLORS.P2} dot={false} strokeWidth={2} />
            <Line type="monotone" dataKey="P3" stroke={SEVERITY_COLORS.P3} dot={false} strokeWidth={1.5} strokeDasharray="4 2" />
          </LineChart>
        </ResponsiveContainer>
        <div className="flex items-center gap-3 mt-2 justify-end">
          {(['P1','P2','P3'] as const).map(s => (
            <span key={s} className="flex items-center gap-1 text-[10px] text-gray-600">
              <span className="w-3 h-0.5 rounded-full inline-block" style={{ background: SEVERITY_COLORS[s] }} />
              {s}
            </span>
          ))}
        </div>
      </div>

      {/* MITRE pie */}
      {tacticData.length > 0 && (
        <div className="bg-white/3 border border-white/5 rounded-xl p-3">
          <p className="text-[10px] text-gray-500 uppercase tracking-widest mb-3">MITRE Tactics</p>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={tacticData}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="45%"
                outerRadius={60}
                innerRadius={28}
                paddingAngle={2}
              >
                {tacticData.map((_, i) => (
                  <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip contentStyle={TOOLTIP_STYLE} />
              <Legend
                wrapperStyle={{ color: TEXT_COLOR, fontSize: 10, paddingTop: 8 }}
                iconSize={8}
                iconType="circle"
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}
