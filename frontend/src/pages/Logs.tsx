import { useState } from 'react'
import { useApi } from '../hooks/useApi'
import { getLogs } from '../lib/api'
import { formatBytes, formatDate } from '../lib/utils'

export default function Logs() {
  const [torrentId, setTorrentId] = useState<number | undefined>()
  const [page, setPage] = useState(0)
  const limit = 50

  const { data: logs, loading } = useApi(
    () => getLogs({ torrentId, limit, offset: page * limit }),
    [torrentId, page]
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Announce Logs</h2>
        <div className="flex gap-3 items-center">
          <input
            type="number"
            placeholder="Filter by Torrent ID"
            value={torrentId ?? ''}
            onChange={(e) => {
              const v = e.target.value ? parseInt(e.target.value) : undefined
              setTorrentId(v)
              setPage(0)
            }}
            className="bg-dark-800 border border-dark-700 rounded px-3 py-1.5 text-sm w-48"
          />
        </div>
      </div>

      <div className="bg-dark-900 border border-dark-800 rounded-lg overflow-x-auto">
        {loading ? (
          <div className="p-8 text-center text-dark-500">Loading...</div>
        ) : !logs || logs.length === 0 ? (
          <div className="p-8 text-center text-dark-500">No logs yet</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-dark-800 text-dark-400">
                <th className="text-left py-3 px-3">Time</th>
                <th className="text-left py-3 px-3">Torrent</th>
                <th className="text-left py-3 px-3">Tracker</th>
                <th className="text-left py-3 px-3">Event</th>
                <th className="text-right py-3 px-3">Upload Delta</th>
                <th className="text-center py-3 px-3">L/S</th>
                <th className="text-center py-3 px-3">Interval</th>
                <th className="text-left py-3 px-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.ID} className="border-b border-dark-800/50 hover:bg-dark-900/50">
                  <td className="py-2 px-3 text-xs text-dark-400">{formatDate(log.Timestamp)}</td>
                  <td className="py-2 px-3">#{log.TorrentID}</td>
                  <td className="py-2 px-3 text-xs text-dark-400 truncate max-w-[200px]">{log.TrackerURL}</td>
                  <td className="py-2 px-3 text-xs">{log.Event || '-'}</td>
                  <td className="py-2 px-3 text-right text-ghost-400">{formatBytes(log.UploadDelta)}</td>
                  <td className="py-2 px-3 text-center text-dark-400">{log.Leechers}/{log.Seeders}</td>
                  <td className="py-2 px-3 text-center text-dark-400">{log.IntervalSecs}s</td>
                  <td className="py-2 px-3">
                    <span className={`text-xs ${log.Status === 'success' ? 'text-green-400' : 'text-red-400'}`}>
                      {log.Status}
                    </span>
                    {log.ErrorMsg && (
                      <span className="text-xs text-red-400 ml-1" title={log.ErrorMsg}>
                        ({log.ErrorMsg.slice(0, 30)})
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="flex gap-2 justify-center">
        <button
          onClick={() => setPage(Math.max(0, page - 1))}
          disabled={page === 0}
          className="px-3 py-1 text-sm bg-dark-800 rounded disabled:opacity-50"
        >
          Previous
        </button>
        <span className="px-3 py-1 text-sm text-dark-400">Page {page + 1}</span>
        <button
          onClick={() => setPage(page + 1)}
          disabled={!logs || logs.length < limit}
          className="px-3 py-1 text-sm bg-dark-800 rounded disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  )
}
