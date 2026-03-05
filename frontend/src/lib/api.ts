import axios from 'axios'
import type { Torrent, StatsOverview, Settings, AnnounceLog, ClientProfile } from './types'

const api = axios.create({
  baseURL: '/api/v1',
})

export async function getTorrents(): Promise<Torrent[]> {
  const { data } = await api.get('/torrents')
  return data
}

export async function addTorrent(file: File): Promise<{ id: number }> {
  const form = new FormData()
  form.append('torrent', file)
  const { data } = await api.post('/torrents', form)
  return data
}

export async function deleteTorrent(id: number): Promise<void> {
  await api.delete(`/torrents/${id}`)
}

export async function startTorrent(id: number): Promise<void> {
  await api.post(`/torrents/${id}/start`)
}

export async function stopTorrent(id: number): Promise<void> {
  await api.post(`/torrents/${id}/stop`)
}

export async function getStatsOverview(): Promise<StatsOverview> {
  const { data } = await api.get('/stats/overview')
  return data
}

export interface StatsHistoryPoint {
  timestamp: string
  totalUploaded: number
  totalLeechers: number
  totalSeeders: number
}

export async function getStatsHistory(hours = 24): Promise<StatsHistoryPoint[]> {
  const { data } = await api.get('/stats/history', { params: { hours } })
  return data
}

export async function getSettings(): Promise<Settings> {
  const { data } = await api.get('/settings')
  return data
}

export async function updateSettings(settings: Partial<Settings>): Promise<Settings> {
  const { data } = await api.put('/settings', settings)
  return data
}

export async function getLogs(params: { torrentId?: number; limit?: number; offset?: number }): Promise<AnnounceLog[]> {
  const { data } = await api.get('/logs', { params })
  return data
}

export async function getRatioTargets(): Promise<Record<string, number>> {
  const { data } = await api.get('/ratio-targets')
  return data
}

export async function updateRatioTargets(targets: Record<string, number>): Promise<void> {
  await api.put('/ratio-targets', targets)
}

export async function getClientProfiles(): Promise<Record<string, ClientProfile>> {
  const { data } = await api.get('/clients/profiles')
  return data
}

export async function refreshProfiles(): Promise<{ profiles: string[] }> {
  const { data } = await api.post('/clients/refresh')
  return data
}

export async function getProwlarrConfig(): Promise<{ url: string; apiKey: string; fetchIntervalMinutes: number }> {
  const { data } = await api.get('/prowlarr/config')
  return data
}

export async function updateProwlarrConfig(config: { url: string; apiKey: string; fetchIntervalMinutes: number }): Promise<void> {
  await api.put('/prowlarr/config', config)
}

export async function triggerProwlarrFetch(): Promise<void> {
  await api.post('/prowlarr/fetch')
}

export interface ProwlarrIndexerFull {
  id: number
  name: string
  protocol: string
  enable: boolean
  implementationName: string
  selected: boolean
  maxUploadSpeedKbs: number | null
  fetchIntervalMinutes: number | null
  maxSlots: number | null
  seedTimeHours: number | null
}

export async function fetchProwlarrIndexers(): Promise<ProwlarrIndexerFull[]> {
  const { data } = await api.post('/prowlarr/indexers')
  return data
}

export async function saveProwlarrIndexers(selections: { id: number; name: string; selected: boolean; maxUploadSpeedKbs: number | null; fetchIntervalMinutes: number | null; maxSlots: number | null; seedTimeHours: number | null }[]): Promise<void> {
  await api.put('/prowlarr/indexers', selections)
}

export async function getSavedProwlarrIndexers(): Promise<{ id: number; name: string; enabled: boolean }[]> {
  const { data } = await api.get('/prowlarr/indexers')
  return data
}
