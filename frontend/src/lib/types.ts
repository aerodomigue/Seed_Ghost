export interface Torrent {
  id: number
  infoHash: string
  name: string
  totalSize: number
  trackerUrl: string
  clientProfile: string
  active: boolean
  status: 'stopped' | 'pending' | 'downloading' | 'seeding' | 'error' | 'deleted'
  errorMsg?: string
  addedAt: string
  source: string
  uploaded: number
  uploadSpeed: number
  downloaded: number
  downloadSpeed: number
  downloadComplete: boolean
  leechers: number
  seeders: number
  indexerId: number | null
  seedTimeRemainingMs: number
  deletedAt?: string
}

export interface StatsOverview {
  totalTorrents: number
  activeTorrents: number
  totalUploaded: number
}

export interface Settings {
  listenAddr: string
  databasePath: string
  profilesDir: string
  defaultClient: string
  autoStart: boolean
  minUploadSpeedKBs: number
  maxUploadSpeedKBs: number
  minDownloadSpeedKBs: number
  maxDownloadSpeedKBs: number
  prowlarrUrl: string
  prowlarrApiKey: string
  fetchIntervalMinutes: number
  prowlarrMaxSlots: number
  logRetentionDays: number
  dataDir: string
}

export interface AnnounceLog {
  ID: number
  TorrentID: number
  Timestamp: string
  TrackerURL: string
  Event: string
  UploadDelta: number
  Leechers: number
  Seeders: number
  IntervalSecs: number
  Status: string
  ErrorMsg: string
}

export interface ClientProfile {
  name: string
  peerIdPrefix: string
  userAgent: string
  numwantDefault: number
}

export interface ProwlarrIndexer {
  id: number
  name: string
  enabled: boolean
  maxUploadSpeedKbs: number | null
}

export interface IndexerStatsPoint {
  indexerId: number | null
  indexerName: string
  timestamp: string
  uploaded: number
  leechers: number
  seeders: number
}
