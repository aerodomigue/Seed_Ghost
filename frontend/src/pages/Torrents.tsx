import { useMemo, useRef, useState } from 'react'
import { usePolling, useApi } from '../hooks/useApi'
import { getTorrents, addTorrent, startTorrent, stopTorrent, deleteTorrent, getSavedProwlarrIndexers } from '../lib/api'
import TorrentList from '../components/TorrentList'
import ConfirmDialog from '../components/ConfirmDialog'

export default function Torrents() {
  const { data: torrents, refresh } = usePolling(getTorrents, 1000)
  const { data: indexers } = useApi(getSavedProwlarrIndexers)
  const indexerMap = useMemo(() => {
    const map: Record<number, string> = {}
    if (indexers) {
      for (const idx of indexers) {
        map[idx.id] = idx.name
      }
    }
    return map
  }, [indexers])
  const fileRef = useRef<HTMLInputElement>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)

  const handleAdd = async () => {
    const file = fileRef.current?.files?.[0]
    if (!file) return
    try {
      await addTorrent(file)
      if (fileRef.current) fileRef.current.value = ''
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to add torrent')
    }
  }

  const handleStart = async (id: number) => {
    try {
      await startTorrent(id)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to start')
    }
  }

  const handleStop = async (id: number) => {
    try {
      await stopTorrent(id)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to stop')
    }
  }

  const handleDeleteRequest = (id: number) => {
    const torrent = torrents?.find(t => t.id === id)
    setDeleteTarget({ id, name: torrent?.name || `Torrent #${id}` })
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    try {
      await deleteTorrent(deleteTarget.id)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete')
    }
    setDeleteTarget(null)
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Torrents</h2>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-gray-400 mb-3">Add Torrent</h3>
        <div className="flex flex-wrap gap-3 items-end">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Torrent File</label>
            <input
              ref={fileRef}
              type="file"
              accept=".torrent"
              className="text-sm file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:bg-gray-800 file:text-gray-300 hover:file:bg-gray-700"
            />
          </div>
          <button
            onClick={handleAdd}
            className="px-4 py-1.5 bg-ghost-600 text-white rounded hover:bg-ghost-700 text-sm font-medium"
          >
            Add
          </button>
        </div>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
        <TorrentList
          torrents={torrents || []}
          indexerMap={indexerMap}
          onStart={handleStart}
          onStop={handleStop}
          onDelete={handleDeleteRequest}
        />
      </div>

      <ConfirmDialog
        open={!!deleteTarget}
        title="Delete Torrent"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
