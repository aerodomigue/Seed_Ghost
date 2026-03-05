import { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'
import {
  getSettings, updateSettings,
  getClientProfiles, refreshProfiles,
  getRatioTargets, updateRatioTargets
} from '../lib/api'

export default function Settings() {
  const { data: settings, refresh: refreshSettings } = useApi(getSettings)
  const { data: profiles, refresh: refreshProfileList } = useApi(getClientProfiles)
  const { data: ratioTargets, refresh: refreshRatios } = useApi(getRatioTargets)

  const [defaultClient, setDefaultClient] = useState('')
  const [autoStart, setAutoStart] = useState(true)
  const [minSpeed, setMinSpeed] = useState(50)
  const [maxSpeed, setMaxSpeed] = useState(5000)
  const [minDlSpeed, setMinDlSpeed] = useState(100)
  const [maxDlSpeed, setMaxDlSpeed] = useState(10000)
  const [fetchInterval, setFetchInterval] = useState(1440)
  const [maxSlots, setMaxSlots] = useState(5)
  const [logRetention, setLogRetention] = useState(7)
  const [saved, setSaved] = useState(false)

  const [newHost, setNewHost] = useState('')
  const [newRatio, setNewRatio] = useState(2.0)

  useEffect(() => {
    if (settings) {
      setDefaultClient(settings.defaultClient)
      setAutoStart(settings.autoStart)
      setMinSpeed(settings.minUploadSpeedKBs)
      setMaxSpeed(settings.maxUploadSpeedKBs)
      setMinDlSpeed(settings.minDownloadSpeedKBs || 100)
      setMaxDlSpeed(settings.maxDownloadSpeedKBs || 10000)
      setFetchInterval(settings.fetchIntervalMinutes || 1440)
      setMaxSlots(settings.prowlarrMaxSlots || 5)
      setLogRetention(settings.logRetentionDays)
    }
  }, [settings])

  const handleSave = async () => {
    try {
      await updateSettings({
        defaultClient,
        autoStart,
        minUploadSpeedKBs: minSpeed,
        maxUploadSpeedKBs: maxSpeed,
        minDownloadSpeedKBs: minDlSpeed,
        maxDownloadSpeedKBs: maxDlSpeed,
        fetchIntervalMinutes: fetchInterval,
        prowlarrMaxSlots: maxSlots,
        logRetentionDays: logRetention,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      refreshSettings()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const handleRefreshProfiles = async () => {
    try {
      await refreshProfiles()
      refreshProfileList()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to refresh')
    }
  }

  const handleAddRatioTarget = async () => {
    if (!newHost) return
    try {
      await updateRatioTargets({ ...ratioTargets, [newHost]: newRatio })
      setNewHost('')
      refreshRatios()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  const profileNames = profiles ? Object.keys(profiles) : []

  return (
    <div className="space-y-6 max-w-2xl">
      <h2 className="text-2xl font-bold">Settings</h2>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-4">
        <h3 className="text-sm font-medium text-gray-400">General</h3>

        <div>
          <label className="block text-xs text-gray-500 mb-1">Default Client Profile</label>
          <div className="flex gap-2">
            <select
              value={defaultClient}
              onChange={(e) => setDefaultClient(e.target.value)}
              className="flex-1 bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            >
              {profileNames.map((name) => (
                <option key={name} value={name}>{name}</option>
              ))}
            </select>
            <button
              onClick={handleRefreshProfiles}
              className="px-3 py-2 bg-gray-800 text-gray-300 rounded hover:bg-gray-700 text-sm"
            >
              Refresh Profiles
            </button>
          </div>
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={autoStart}
            onChange={(e) => setAutoStart(e.target.checked)}
            className="rounded bg-gray-800 border-gray-700"
          />
          Auto-start torrents when added
        </label>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Min Upload Speed (KB/s)</label>
            <input
              type="number"
              value={minSpeed}
              onChange={(e) => setMinSpeed(parseInt(e.target.value) || 0)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Max Upload Speed (KB/s)</label>
            <input
              type="number"
              value={maxSpeed}
              onChange={(e) => setMaxSpeed(parseInt(e.target.value) || 0)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Min Download Speed (KB/s)</label>
            <input
              type="number"
              value={minDlSpeed}
              onChange={(e) => setMinDlSpeed(parseInt(e.target.value) || 0)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Max Download Speed (KB/s)</label>
            <input
              type="number"
              value={maxDlSpeed}
              onChange={(e) => setMaxDlSpeed(parseInt(e.target.value) || 0)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Prowlarr Fetch Interval (minutes)</label>
            <input
              type="number"
              value={fetchInterval}
              onChange={(e) => setFetchInterval(parseInt(e.target.value) || 1440)}
              min={5}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Default Max Active Torrents</label>
            <input
              type="number"
              value={maxSlots}
              onChange={(e) => setMaxSlots(parseInt(e.target.value) || 5)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Log Retention (days)</label>
            <input
              type="number"
              value={logRetention}
              onChange={(e) => setLogRetention(parseInt(e.target.value) || 7)}
              min={1}
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
        </div>

        <button
          onClick={handleSave}
          className="px-4 py-2 bg-ghost-600 text-white rounded hover:bg-ghost-700 text-sm font-medium"
        >
          {saved ? 'Saved!' : 'Save Settings'}
        </button>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-4">
        <h3 className="text-sm font-medium text-gray-400">Ratio Targets</h3>
        <p className="text-xs text-gray-500">Set a target ratio per tracker. Seeding stops when the target is reached.</p>

        {ratioTargets && Object.keys(ratioTargets).length > 0 && (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2">Tracker Host</th>
                <th className="text-right py-2">Target Ratio</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(ratioTargets).map(([host, ratio]) => (
                <tr key={host} className="border-b border-gray-800/50">
                  <td className="py-2">{host}</td>
                  <td className="py-2 text-right font-mono">{ratio}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">Tracker Host</label>
            <input
              type="text"
              value={newHost}
              onChange={(e) => setNewHost(e.target.value)}
              placeholder="tracker.example.com"
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Ratio</label>
            <input
              type="number"
              value={newRatio}
              onChange={(e) => setNewRatio(parseFloat(e.target.value) || 2.0)}
              step={0.5}
              min={0.1}
              className="w-24 bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>
          <button
            onClick={handleAddRatioTarget}
            className="px-3 py-2 bg-gray-800 text-gray-300 rounded hover:bg-gray-700 text-sm"
          >
            Add
          </button>
        </div>
      </div>
    </div>
  )
}
