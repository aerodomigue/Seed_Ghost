import { useState } from 'react'
import { useApi } from '../hooks/useApi'
import { getProwlarrConfig, updateProwlarrConfig, triggerProwlarrFetch } from '../lib/api'

export default function Prowlarr() {
  const { data: config, refresh } = useApi(getProwlarrConfig)
  const [url, setUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [fetchInterval, setFetchInterval] = useState(30)
  const [saved, setSaved] = useState(false)

  // Initialize form when config loads
  useState(() => {
    if (config) {
      setUrl(config.url)
      setApiKey(config.apiKey)
      setFetchInterval(config.fetchIntervalMinutes)
    }
  })

  const handleSave = async () => {
    try {
      await updateProwlarrConfig({ url, apiKey, fetchIntervalMinutes: fetchInterval })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      refresh()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const handleFetch = async () => {
    try {
      await triggerProwlarrFetch()
      alert('Fetch triggered')
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Fetch failed')
    }
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Prowlarr Integration</h2>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-4 max-w-xl">
        <h3 className="text-sm font-medium text-gray-400">Configuration</h3>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Prowlarr URL</label>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="http://localhost:9696"
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="Enter Prowlarr API key"
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Fetch Interval (minutes)</label>
          <input
            type="number"
            value={fetchInterval}
            onChange={(e) => setFetchInterval(parseInt(e.target.value) || 30)}
            min={5}
            className="w-32 bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
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
            className="px-4 py-2 bg-gray-800 text-gray-300 rounded hover:bg-gray-700 text-sm font-medium"
          >
            Trigger Fetch Now
          </button>
        </div>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6">
        <h3 className="text-sm font-medium text-gray-400 mb-3">Indexers</h3>
        <p className="text-gray-500 text-sm">
          Configure Prowlarr URL and API key above to see available indexers.
        </p>
      </div>
    </div>
  )
}
