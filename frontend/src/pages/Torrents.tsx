import { useState, useRef } from 'react'
import { usePolling } from '../hooks/useApi'
import { getTorrents, addTorrent, startTorrent, stopTorrent, deleteTorrent, getClientProfiles } from '../lib/api'
import { useApi } from '../hooks/useApi'
import TorrentList from '../components/TorrentList'

export default function Torrents() {
  const { data: torrents, refresh } = usePolling(getTorrents, 5000)
  const { data: profiles } = useApi(getClientProfiles)
  const [selectedProfile, setSelectedProfile] = useState('')
  const [autoStart, setAutoStart] = useState(true)
  const fileRef = useRef<HTMLInputElement>(null)

  const handleAdd = async () => {
    const file = fileRef.current?.files?.[0]
    if (!file) return
    try {
      await addTorrent(file, selectedProfile, autoStart)
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

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this torrent?')) return
    try {
      await deleteTorrent(id)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete')
    }
  }

  const profileNames = profiles ? Object.keys(profiles) : []

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
          <div>
            <label className="block text-xs text-gray-500 mb-1">Client Profile</label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
            >
              <option value="">Default</option>
              {profileNames.map((name) => (
                <option key={name} value={name}>{name}</option>
              ))}
            </select>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={autoStart}
              onChange={(e) => setAutoStart(e.target.checked)}
              className="rounded bg-gray-800 border-gray-700"
            />
            Auto-start
          </label>
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
          onStart={handleStart}
          onStop={handleStop}
          onDelete={handleDelete}
        />
      </div>
    </div>
  )
}
