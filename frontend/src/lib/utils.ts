export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1)
  if (i <= 0) return `${Math.floor(bytes)} B`
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

export function formatSpeed(bytesPerSec: number): string {
  return `${formatBytes(bytesPerSec)}/s`
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}

export function classNames(...classes: (string | false | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}

export function formatSeedTime(remainingMs: number): string {
  const absMs = Math.abs(remainingMs)
  const totalSeconds = Math.floor(absMs / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)

  let time: string
  if (days > 0) time = `${days}d ${hours}h ${minutes}m`
  else if (hours > 0) time = `${hours}h ${minutes}m`
  else time = `${minutes}m`

  // Positive remaining = still counting down (prefix with -)
  // Zero or negative = surplus (prefix with +)
  return remainingMs > 0 ? `-${time}` : `+${time}`
}
