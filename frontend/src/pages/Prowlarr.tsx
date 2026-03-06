import { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'
import {
  getProwlarrConfig, updateProwlarrConfig, triggerProwlarrFetch,
  fetchProwlarrIndexers, saveProwlarrIndexers,
  type ProwlarrIndexerFull
} from '../lib/api'

export default function Prowlarr() {
  const { data: config, refresh } = useApi(getProwlarrConfig)
  const [url, setUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (config) {
      setUrl(config.url)
      setApiKey(config.apiKey)
    }
  }, [config])

  const handleSave = async () => {
    try {
      await updateProwlarrConfig({ url, apiKey, fetchIntervalMinutes: config?.fetchIntervalMinutes ?? 1440 })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  // --- Fetch torrents ---
  const [fetching, setFetching] = useState(false)
  const [fetchError, setFetchError] = useState('')

  const handleFetch = async () => {
    setFetching(true)
    setFetchError('')
    try {
      await triggerProwlarrFetch()
    } catch (err) {
      if (err instanceof Error) {
        const msg = (err as { response?: { data?: { error?: string } } }).response?.data?.error || err.message
        setFetchError(msg)
      }
    } finally {
      setFetching(false)
    }
  }

  // --- Indexers ---
  const [indexers, setIndexers] = useState<ProwlarrIndexerFull[]>([])
  const [loadingIndexers, setLoadingIndexers] = useState(false)
  const [indexerError, setIndexerError] = useState('')
  const [indexersSaved, setIndexersSaved] = useState(false)

  // Auto-load indexers when Prowlarr is configured
  useEffect(() => {
    if (config?.url && config?.apiKey) {
      handleLoadIndexers()
    }
  }, [config?.url, config?.apiKey]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleLoadIndexers = async () => {
    setLoadingIndexers(true)
    setIndexerError('')
    try {
      const data = await fetchProwlarrIndexers()
      setIndexers(data)
    } catch (err) {
      const msg = (err as { response?: { data?: { error?: string } } }).response?.data?.error
        || (err instanceof Error ? err.message : 'Failed to load indexers')
      setIndexerError(msg)
    } finally {
      setLoadingIndexers(false)
    }
  }

  const toggleIndexer = (id: number) => {
    setIndexers(prev => prev.map(idx =>
      idx.id === id ? { ...idx, selected: !idx.selected } : idx
    ))
  }

  const setIndexerSpeed = (id: number, speed: number | null) => {
    setIndexers(prev => prev.map(idx =>
      idx.id === id ? { ...idx, maxUploadSpeedKbs: speed } : idx
    ))
  }

  const setIndexerFetchInterval = (id: number, minutes: number | null) => {
    setIndexers(prev => prev.map(idx =>
      idx.id === id ? { ...idx, fetchIntervalMinutes: minutes } : idx
    ))
  }

  const setIndexerMaxSlots = (id: number, slots: number | null) => {
    setIndexers(prev => prev.map(idx =>
      idx.id === id ? { ...idx, maxSlots: slots } : idx
    ))
  }

  const setIndexerSeedTime = (id: number, hours: number | null) => {
    setIndexers(prev => prev.map(idx =>
      idx.id === id ? { ...idx, seedTimeHours: hours } : idx
    ))
  }

  const handleSaveIndexers = async () => {
    try {
      await saveProwlarrIndexers(indexers.map(idx => ({
        id: idx.id,
        name: idx.name,
        selected: idx.selected,
        maxUploadSpeedKbs: idx.maxUploadSpeedKbs,
        fetchIntervalMinutes: idx.fetchIntervalMinutes,
        maxSlots: idx.maxSlots,
        seedTimeHours: idx.seedTimeHours,
      })))
      setIndexersSaved(true)
      setTimeout(() => setIndexersSaved(false), 2000)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const selectedCount = indexers.filter(i => i.selected).length

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Prowlarr Integration</h2>

      <div className="bg-dark-900 border border-dark-800 rounded-lg p-6 space-y-4 max-w-xl">
        <h3 className="text-sm font-medium text-dark-400">Configuration</h3>
        <div>
          <label className="block text-xs text-dark-500 mb-1">Prowlarr URL</label>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="http://localhost:9696"
            className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="block text-xs text-dark-500 mb-1">API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="Enter Prowlarr API key"
            className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2 text-sm"
          />
        </div>
        <div className="flex gap-3">
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-ghost-600 text-white rounded hover:bg-ghost-700 text-sm font-medium"
          >
            {saved ? 'Saved!' : 'Save'}
          </button>
          <button
            onClick={handleFetch}
            disabled={fetching}
            className="px-4 py-2 bg-dark-800 text-dark-300 rounded hover:bg-dark-700 text-sm font-medium disabled:opacity-50"
          >
            {fetching ? 'Syncing...' : 'Sync Torrents'}
          </button>
        </div>
        {fetchError && (
          <p className="text-sm text-red-400">{fetchError}</p>
        )}
      </div>

      <div className="bg-dark-900 border border-dark-800 rounded-lg p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-dark-400">Indexers</h3>
            {indexers.length > 0 && (
              <p className="text-xs text-dark-500 mt-1">{selectedCount} selected out of {indexers.length}</p>
            )}
          </div>
          <button
            onClick={handleLoadIndexers}
            disabled={loadingIndexers}
            className="px-3 py-1.5 bg-ghost-600 text-white rounded hover:bg-ghost-700 text-sm font-medium disabled:opacity-50"
          >
            {loadingIndexers ? 'Loading...' : indexers.length > 0 ? 'Refresh from Prowlarr' : 'Load from Prowlarr'}
          </button>
        </div>

        {indexerError && (
          <p className="text-sm text-red-400">{indexerError}</p>
        )}

        {indexers.length === 0 && !indexerError && (
          <p className="text-dark-500 text-sm">
            Click "Load from Prowlarr" to fetch available indexers.
          </p>
        )}

        {indexers.length > 0 && (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-dark-800 text-dark-400">
                    <th className="text-left py-2 px-2 w-10">Use</th>
                    <th className="text-left py-2 px-2 w-14">ID</th>
                    <th className="text-left py-2 px-2">Name</th>
                    <th className="text-left py-2 px-2">Protocol</th>
                    <th className="text-left py-2 px-2">Type</th>
                    <th className="text-center py-2 px-2">Prowlarr</th>
                    <th className="text-left py-2 px-2">Max Speed (KB/s)</th>
                    <th className="text-left py-2 px-2">Fetch Interval (min)</th>
                    <th className="text-left py-2 px-2">Max Torrents</th>
                    <th className="text-left py-2 px-2">H&R Time (h)</th>
                  </tr>
                </thead>
                <tbody>
                  {indexers.map((idx) => (
                    <tr key={idx.id} className="border-b border-dark-800/50 hover:bg-dark-900/50">
                      <td className="py-2 px-2">
                        <input
                          type="checkbox"
                          checked={idx.selected}
                          onChange={() => toggleIndexer(idx.id)}
                          className="rounded bg-dark-800 border-dark-700"
                        />
                      </td>
                      <td className="py-2 px-2 text-dark-500 font-mono text-xs">{idx.id}</td>
                      <td className="py-2 px-2 font-medium">{idx.name}</td>
                      <td className="py-2 px-2 text-dark-400">{idx.protocol}</td>
                      <td className="py-2 px-2 text-dark-400 text-xs">{idx.implementationName}</td>
                      <td className="py-2 px-2 text-center">
                        <span className={`inline-block w-2 h-2 rounded-full ${idx.enable ? 'bg-green-400' : 'bg-dark-600'}`} />
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="number"
                          value={idx.maxUploadSpeedKbs ?? ''}
                          onChange={(e) => setIndexerSpeed(idx.id, e.target.value ? parseInt(e.target.value) : null)}
                          placeholder="Default"
                          disabled={!idx.selected}
                          className="w-24 bg-dark-800 border border-dark-700 rounded px-2 py-1 text-xs disabled:opacity-40"
                        />
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="number"
                          value={idx.fetchIntervalMinutes ?? ''}
                          onChange={(e) => setIndexerFetchInterval(idx.id, e.target.value ? parseInt(e.target.value) : null)}
                          placeholder="Default"
                          disabled={!idx.selected}
                          className="w-24 bg-dark-800 border border-dark-700 rounded px-2 py-1 text-xs disabled:opacity-40"
                        />
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="number"
                          value={idx.maxSlots ?? ''}
                          onChange={(e) => setIndexerMaxSlots(idx.id, e.target.value ? parseInt(e.target.value) : null)}
                          placeholder="Default"
                          disabled={!idx.selected}
                          className="w-24 bg-dark-800 border border-dark-700 rounded px-2 py-1 text-xs disabled:opacity-40"
                        />
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="number"
                          value={idx.seedTimeHours ?? ''}
                          onChange={(e) => setIndexerSeedTime(idx.id, e.target.value ? parseInt(e.target.value) : null)}
                          placeholder="72"
                          disabled={!idx.selected}
                          className="w-24 bg-dark-800 border border-dark-700 rounded px-2 py-1 text-xs disabled:opacity-40"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <button
              onClick={handleSaveIndexers}
              className="px-4 py-2 bg-ghost-600 text-white rounded hover:bg-ghost-700 text-sm font-medium"
            >
              {indexersSaved ? 'Saved!' : 'Save Selection'}
            </button>
          </>
        )}
      </div>
    </div>
  )
}
