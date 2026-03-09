import { useMemo, useRef, useState } from 'react'
import { usePolling, useApi } from '../hooks/useApi'
import { getTorrents, addTorrent, startTorrent, stopTorrent, deleteTorrent, getDeletedTorrents, getSavedProwlarrIndexers } from '../lib/api'
import TorrentList from '../components/TorrentList'
import ConfirmDialog from '../components/ConfirmDialog'

export default function Torrents() {
  const [tab, setTab] = useState<'active' | 'history'>('active')
  const { data: torrents, refresh } = usePolling(getTorrents, 1000)
  const { data: deletedTorrents, refresh: refreshDeleted } = usePolling(getDeletedTorrents, 5000)
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
      refreshDeleted()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete')
    }
    setDeleteTarget(null)
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Torrents</h2>

      {tab === 'active' && (
        <div className="bg-dark-900 border border-dark-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-dark-400 mb-3">Add Torrent</h3>
          <div className="flex flex-wrap gap-3 items-end">
            <div>
              <label className="block text-xs text-dark-500 mb-1">Torrent File</label>
              <input
                ref={fileRef}
                type="file"
                accept=".torrent"
                className="text-sm file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:bg-dark-800 file:text-dark-300 hover:file:bg-dark-700"
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
      )}

      <div className="bg-dark-900 border border-dark-800 rounded-lg p-4">
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => setTab('active')}
            className={`px-3 py-1.5 text-sm font-medium rounded ${
              tab === 'active'
                ? 'bg-ghost-600 text-white'
                : 'bg-dark-800 text-dark-400 hover:text-dark-300'
            }`}
          >
            Active
          </button>
          <button
            onClick={() => setTab('history')}
            className={`px-3 py-1.5 text-sm font-medium rounded ${
              tab === 'history'
                ? 'bg-ghost-600 text-white'
                : 'bg-dark-800 text-dark-400 hover:text-dark-300'
            }`}
          >
            History
          </button>
        </div>

        {tab === 'active' ? (
          <TorrentList
            torrents={torrents || []}
            indexerMap={indexerMap}
            onStart={handleStart}
            onStop={handleStop}
            onDelete={handleDeleteRequest}
          />
        ) : (
          <TorrentList
            torrents={deletedTorrents || []}
            indexerMap={indexerMap}
            onStart={handleStart}
            onStop={handleStop}
            onDelete={handleDeleteRequest}
            readonly
          />
        )}
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
