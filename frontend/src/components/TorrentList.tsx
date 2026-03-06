import type { Torrent } from '../lib/types'
import { formatBytes, formatSeedTime } from '../lib/utils'

interface TorrentListProps {
  torrents: Torrent[]
  indexerMap: Record<number, string>
  onStart: (id: number) => void
  onStop: (id: number) => void
  onDelete: (id: number) => void
}

export default function TorrentList({ torrents, indexerMap, onStart, onStop, onDelete }: TorrentListProps) {
  if (torrents.length === 0) {
    return (
      <div className="text-center py-12 text-dark-500">
        No torrents added yet. Upload a .torrent file to get started.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-dark-800 text-dark-400">
            <th className="text-left py-3 px-2">Name</th>
            <th className="text-left py-3 px-2 w-20">Size</th>
            <th className="text-left py-3 px-2 w-20">Uploaded</th>
            <th className="text-left py-3 px-2 w-24">UL Speed</th>
            <th className="text-left py-3 px-2 w-20">Downloaded</th>
            <th className="text-left py-3 px-2 w-24">DL Speed</th>
            <th className="text-center py-3 px-2 w-16">Ratio</th>
            <th className="text-center py-3 px-2 w-14">L/S</th>
            <th className="text-left py-3 px-2 w-24">Source</th>
            <th className="text-left py-3 px-2 w-28">Client</th>
            <th className="text-left py-3 px-2 w-20">Seed Time</th>
            <th className="text-left py-3 px-2 w-20">Status</th>
            <th className="text-right py-3 px-2 w-24">Actions</th>
          </tr>
        </thead>
        <tbody>
          {torrents.map((t) => {
            const ratio = t.totalSize > 0 ? (t.uploaded / t.totalSize).toFixed(2) : '0.00'
            return (
              <tr key={t.id} className="border-b border-dark-800/50 hover:bg-dark-900/50">
                <td className="py-3 px-2">
                  <div className="font-medium truncate max-w-xs" title={t.name}>{t.name}</div>
                  <div className="text-xs text-dark-500 truncate max-w-xs" title={t.trackerUrl}>
                    {new URL(t.trackerUrl).hostname}
                  </div>
                </td>
                <td className="py-3 px-2 text-dark-400 whitespace-nowrap">{formatBytes(t.totalSize)}</td>
                <td className="py-3 px-2 text-ghost-400 whitespace-nowrap">{formatBytes(t.uploaded)}</td>
                <td className="py-3 px-2 text-dark-400 whitespace-nowrap">
                  {t.active && t.uploadSpeed > 0 ? `${formatBytes(t.uploadSpeed)}/s` : '-'}
                </td>
                <td className="py-3 px-2 text-blue-400 whitespace-nowrap">
                  {t.downloadComplete ? formatBytes(t.totalSize) : formatBytes(t.downloaded)}
                </td>
                <td className="py-3 px-2 text-dark-400 whitespace-nowrap">
                  {t.active && !t.downloadComplete && t.downloadSpeed > 0 ? `${formatBytes(t.downloadSpeed)}/s` : '-'}
                </td>
                <td className="py-3 px-2 text-center whitespace-nowrap">
                  <span className={`font-mono ${parseFloat(ratio) >= 1 ? 'text-green-400' : 'text-yellow-400'}`}>
                    {ratio}
                  </span>
                </td>
                <td className="py-3 px-2 text-center text-dark-400 whitespace-nowrap">
                  {t.leechers}/{t.seeders}
                </td>
                <td className="py-3 px-2 text-xs">
                  {t.indexerId != null ? (
                    <span className="text-ghost-400" title={`Indexer #${t.indexerId}`}>
                      {indexerMap[t.indexerId] || `#${t.indexerId}`}
                    </span>
                  ) : (
                    <span className="text-dark-500">Manual</span>
                  )}
                </td>
                <td className="py-3 px-2 text-dark-400 text-xs">{t.clientProfile || '-'}</td>
                <td className="py-3 px-2 text-xs">
                  <span className={t.seedTimeRemainingMs <= 0 ? 'text-green-400' : 'text-orange-400'}>
                    {formatSeedTime(t.seedTimeRemainingMs)}
                  </span>
                </td>
                <td className="py-3 px-2">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                    t.status === 'seeding'
                      ? 'bg-green-900/50 text-green-400'
                      : t.status === 'downloading'
                      ? 'bg-blue-900/50 text-blue-400'
                      : t.status === 'pending'
                      ? 'bg-yellow-900/50 text-yellow-400'
                      : 'bg-dark-800 text-dark-400'
                  }`}>
                    {t.status === 'seeding' ? 'Seeding' : t.status === 'downloading' ? 'Downloading' : t.status === 'pending' ? 'Pending' : 'Stopped'}
                  </span>
                </td>
                <td className="py-3 px-2 text-right">
                  <div className="flex gap-1 justify-end">
                    {t.active ? (
                      <button
                        onClick={() => onStop(t.id)}
                        className="px-2 py-1 text-xs bg-yellow-900/50 text-yellow-400 rounded hover:bg-yellow-900"
                      >
                        Stop
                      </button>
                    ) : (
                      <button
                        onClick={() => onStart(t.id)}
                        className="px-2 py-1 text-xs bg-green-900/50 text-green-400 rounded hover:bg-green-900"
                      >
                        Start
                      </button>
                    )}
                    <button
                      onClick={() => onDelete(t.id)}
                      className="px-2 py-1 text-xs bg-red-900/50 text-red-400 rounded hover:bg-red-900"
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
