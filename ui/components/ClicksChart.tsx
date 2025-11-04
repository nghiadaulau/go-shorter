import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts'

export type ClickPoint = { day: string; total: number }

function formatDayLabel(day: string) {
  if (!day) return ''
  if (day.includes('T')) {
    const d = new Date(day)
    if (!isNaN(d.getTime())) {
      const mm = String(d.getUTCMonth() + 1).padStart(2, '0')
      const dd = String(d.getUTCDate()).padStart(2, '0')
      const hh = String(d.getUTCHours()).padStart(2, '0')
      const mi = String(d.getUTCMinutes()).padStart(2, '0')
      const ss = String(d.getUTCSeconds()).padStart(2, '0')
      if (ss !== '00') {
        return `${mm}-${dd} ${hh}:${mi}:${ss}`
      }
      return `${mm}-${dd} ${hh}:${mi}`
    }
  }
  return day.slice(5)
}

export default function ClicksChart({ points }: { points: ClickPoint[] }) {
  const data = (points || []).map(p => ({ name: formatDayLabel(p.day), value: p.total || 0 }))

  return (
    <div className="w-full h-64">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 10, right: 16, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="colorClicks" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.35}/>
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.03}/>
            </linearGradient>
          </defs>
          <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" />
          <XAxis dataKey="name" tick={{ fontSize: 12, fill: '#6b7280' }} axisLine={{ stroke: '#e5e7eb' }} tickLine={false} />
          <YAxis allowDecimals={false} tick={{ fontSize: 12, fill: '#6b7280' }} axisLine={{ stroke: '#e5e7eb' }} tickLine={false} />
          <Tooltip
            contentStyle={{ backgroundColor: 'white', border: '1px solid #e5e7eb', borderRadius: 8 }}
            labelStyle={{ color: '#111827' }}
            itemStyle={{ color: '#1f2937' }}
          />
          <Area type="monotone" dataKey="value" stroke="#2563eb" strokeWidth={2} fillOpacity={1} fill="url(#colorClicks)" dot={false} activeDot={{ r: 4 }} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}


