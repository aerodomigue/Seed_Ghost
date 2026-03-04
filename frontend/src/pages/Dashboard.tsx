import { usePolling } from '../hooks/useApi'
import { getStatsOverview, getTorrents } from '../lib/api'
import { formatBytes } from '../lib/utils'
import StatsCard from '../components/StatsCard'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { useState, useEffect } from 'react'

export default function Dashboard() {
  const { data: stats } = usePolling(getStatsOverview, 5000)
  const { data: torrents } = usePolling(getTorrents, 5000)
  const [uploadHistory, setUploadHistory] = useState<{ time: string; uploaded: number }[]>([])

  useEffect(() => {
    if (stats) {
      setUploadHistory((prev) => {
        const next = [...prev, { time: new Date().toLocaleTimeString(), uploaded: stats.totalUploaded }]
        return next.slice(-30) // Keep last 30 data points
      })
    }
  }, [stats])

  const activeTorrents = torrents?.filter((t) => t.active) || []
  const totalLeechers = activeTorrents.reduce((sum, t) => sum + t.leechers, 0)

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Dashboard</h2>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          label="Total Torrents"
          value={String(stats?.totalTorrents ?? 0)}
          sub={`${stats?.activeTorrents ?? 0} active`}
        />
        <StatsCard
          label="Total Uploaded"
          value={formatBytes(stats?.totalUploaded ?? 0)}
        />
        <StatsCard
          label="Active Leechers"
          value={String(totalLeechers)}
          sub="across all torrents"
        />
        <StatsCard
          label="Active Sessions"
          value={String(stats?.activeTorrents ?? 0)}
        />
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-gray-400 mb-4">Upload Progress</h3>
        <div className="h-64">
          {uploadHistory.length > 1 ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={uploadHistory}>
                <XAxis dataKey="time" tick={{ fill: '#6b7280', fontSize: 12 }} />
                <YAxis
                  tick={{ fill: '#6b7280', fontSize: 12 }}
                  tickFormatter={(v: number) => formatBytes(v)}
                />
                <Tooltip
                  contentStyle={{ backgroundColor: '#111827', border: '1px solid #374151', borderRadius: 8 }}
                  labelStyle={{ color: '#9ca3af' }}
                  formatter={(v: number) => [formatBytes(v), 'Uploaded']}
                />
                <Area type="monotone" dataKey="uploaded" stroke="#0ea5e9" fill="#0ea5e9" fillOpacity={0.1} />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              Collecting data...
            </div>
          )}
        </div>
      </div>

      {activeTorrents.length > 0 && (
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-400 mb-3">Active Torrents</h3>
          <div className="space-y-2">
            {activeTorrents.map((t) => (
              <div key={t.id} className="flex items-center justify-between py-2 border-b border-gray-800/50 last:border-0">
                <div className="truncate max-w-md">
                  <span className="text-sm">{t.name}</span>
                </div>
                <div className="flex gap-4 text-sm text-gray-400">
                  <span>Up: {formatBytes(t.uploaded)}</span>
                  <span>L/S: {t.leechers}/{t.seeders}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
