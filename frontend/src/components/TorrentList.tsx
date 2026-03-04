import type { Torrent } from '../lib/types'
import { formatBytes } from '../lib/utils'

interface TorrentListProps {
  torrents: Torrent[]
  onStart: (id: number) => void
  onStop: (id: number) => void
  onDelete: (id: number) => void
}

export default function TorrentList({ torrents, onStart, onStop, onDelete }: TorrentListProps) {
  if (torrents.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        No torrents added yet. Upload a .torrent file to get started.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-800 text-gray-400">
            <th className="text-left py-3 px-2">Name</th>
            <th className="text-left py-3 px-2">Size</th>
            <th className="text-left py-3 px-2">Uploaded</th>
            <th className="text-center py-3 px-2">Ratio</th>
            <th className="text-center py-3 px-2">L/S</th>
            <th className="text-left py-3 px-2">Client</th>
            <th className="text-left py-3 px-2">Status</th>
            <th className="text-right py-3 px-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {torrents.map((t) => {
            const ratio = t.totalSize > 0 ? (t.uploaded / t.totalSize).toFixed(2) : '0.00'
            return (
              <tr key={t.id} className="border-b border-gray-800/50 hover:bg-gray-900/50">
                <td className="py-3 px-2">
                  <div className="font-medium truncate max-w-xs" title={t.name}>{t.name}</div>
                  <div className="text-xs text-gray-500 truncate max-w-xs" title={t.trackerUrl}>
                    {new URL(t.trackerUrl).hostname}
                  </div>
                </td>
                <td className="py-3 px-2 text-gray-400">{formatBytes(t.totalSize)}</td>
                <td className="py-3 px-2 text-ghost-400">{formatBytes(t.uploaded)}</td>
                <td className="py-3 px-2 text-center">
                  <span className={`font-mono ${parseFloat(ratio) >= 1 ? 'text-green-400' : 'text-yellow-400'}`}>
                    {ratio}
                  </span>
                </td>
                <td className="py-3 px-2 text-center text-gray-400">
                  {t.leechers}/{t.seeders}
                </td>
                <td className="py-3 px-2 text-gray-400 text-xs">{t.clientProfile || '-'}</td>
                <td className="py-3 px-2">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                    t.active
                      ? 'bg-green-900/50 text-green-400'
                      : 'bg-gray-800 text-gray-400'
                  }`}>
                    {t.active ? 'Seeding' : 'Stopped'}
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
