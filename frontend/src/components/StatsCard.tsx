interface StatsCardProps {
  label: string
  value: string
  sub?: string
}

export default function StatsCard({ label, value, sub }: StatsCardProps) {
  return (
    <div className="bg-dark-900 border border-dark-800 rounded-lg p-4">
      <p className="text-sm text-dark-400">{label}</p>
      <p className="text-2xl font-bold text-dark-100 mt-1">{value}</p>
      {sub && <p className="text-xs text-dark-500 mt-1">{sub}</p>}
    </div>
  )
}
