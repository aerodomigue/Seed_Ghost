import { useMemo } from 'react'
import { usePolling } from '../hooks/useApi'
import { getStatsOverview, getTorrents, getStatsHistoryByIndexer } from '../lib/api'
import { formatBytes, hashColor } from '../lib/utils'
import StatsCard from '../components/StatsCard'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import type { IndexerStatsPoint } from '../lib/types'

function buildChartData(points: IndexerStatsPoint[]) {
  const byTime = new Map<string, Record<string, number>>()
  for (const p of points) {
    const time = new Date(p.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    let row = byTime.get(time)
    if (!row) {
      row = { time: 0 } as unknown as Record<string, number>
      byTime.set(time, row)
    }
    row[p.indexerName] = p.uploaded
  }
  return Array.from(byTime.entries()).map(([time, row]) => ({ ...row, time }))
}

function getIndexerNames(points: IndexerStatsPoint[]): string[] {
  const seen = new Set<string>()
  for (const p of points) seen.add(p.indexerName)
  return Array.from(seen).sort()
}

export default function Dashboard() {
  const { data: stats } = usePolling(getStatsOverview, 5000)
  const { data: torrents } = usePolling(getTorrents, 1000)
  const { data: indexerHistory } = usePolling(() => getStatsHistoryByIndexer(24), 10000)

  const activeTorrents = torrents?.filter((t) => t.active) || []
  const totalLeechers = activeTorrents.reduce((sum, t) => sum + t.leechers, 0)
  const totalSpeed = activeTorrents.reduce((sum, t) => sum + (t.uploadSpeed ?? 0), 0)

  const indexerNames = useMemo(() => getIndexerNames(indexerHistory || []), [indexerHistory])
  const chartData = useMemo(() => buildChartData(indexerHistory || []), [indexerHistory])
  const colors = useMemo(() => {
    const map: Record<string, string> = {}
    for (const name of indexerNames) map[name] = hashColor(name)
    return map
  }, [indexerNames])

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
          label="Upload Speed"
          value={`${formatBytes(totalSpeed)}/s`}
          sub={`${stats?.activeTorrents ?? 0} active sessions`}
        />
      </div>

      <div className="bg-dark-900 border border-dark-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-dark-400 mb-4">Upload Progress (24h)</h3>
        <div className="h-64">
          {chartData.length > 1 ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <XAxis dataKey="time" tick={{ fill: '#5a756a', fontSize: 12 }} />
                <YAxis
                  tick={{ fill: '#5a756a', fontSize: 12 }}
                  tickFormatter={(v: number) => formatBytes(v)}
                />
                <Tooltip
                  contentStyle={{ backgroundColor: '#111916', border: '1px solid #25342d', borderRadius: 8 }}
                  labelStyle={{ color: '#829a90' }}
                  formatter={(v: number, name: string) => [formatBytes(v), name]}
                />
                <Legend
                  wrapperStyle={{ fontSize: 12, color: '#829a90' }}
                />
                {indexerNames.map((name) => (
                  <Area
                    key={name}
                    type="monotone"
                    dataKey={name}
                    stroke={colors[name]}
                    fill={colors[name]}
                    fillOpacity={0.1}
                    connectNulls
                  />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-dark-500">
              Collecting data...
            </div>
          )}
        </div>
      </div>

      {activeTorrents.length > 0 && (
        <div className="bg-dark-900 border border-dark-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-400 mb-3">Active Torrents</h3>
          <div className="space-y-2">
            {activeTorrents.map((t) => (
              <div key={t.id} className="flex items-center justify-between py-2 border-b border-dark-800/50 last:border-0">
                <div className="truncate max-w-md">
                  <span className="text-sm">{t.name}</span>
                </div>
                <div className="flex gap-4 text-sm text-dark-400">
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
