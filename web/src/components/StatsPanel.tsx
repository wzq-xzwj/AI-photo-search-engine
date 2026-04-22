import { useState, useEffect } from 'react'

interface Stats {
  total_photos: number
  total_size: number
  vector_images: number
  total_tags: number
}

export default function StatsPanel() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    fetch('/api/v1/stats')
      .then(res => {
        if (!res.ok) throw new Error('stats failed')
        return res.json()
      })
      .then(data => {
        setStats(data)
        setError(false)
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="card dark:card-dark p-4 animate-pulse">
            <div className="w-10 h-10 rounded-xl bg-gray-200 dark:bg-dark-surface mb-3" />
            <div className="h-7 w-16 bg-gray-200 dark:bg-dark-surface rounded mb-1" />
            <div className="h-4 w-24 bg-gray-200 dark:bg-dark-surface rounded" />
          </div>
        ))}
      </div>
    )
  }

  if (error || !stats) {
    return (
      <div className="card dark:card-dark p-6 text-center">
        <p className="text-sm text-gray-500 dark:text-dark-muted">
          无法加载统计数据
        </p>
      </div>
    )
  }

  const rawSize = stats.total_size ?? 0
  const formattedSize = rawSize > 1073741824
    ? `${(rawSize / 1073741824).toFixed(1)} GB`
    : rawSize > 1048576
      ? `${(rawSize / 1048576).toFixed(1)} MB`
      : `${(rawSize / 1024).toFixed(0)} KB`

  const cards = [
    {
      label: '照片总数',
      value: (stats.total_photos ?? 0).toLocaleString(),
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
        </svg>
      ),
      color: 'from-primary-500 to-primary-600',
    },
    {
      label: '向量索引',
      value: (stats.vector_images ?? 0).toString(),
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
        </svg>
      ),
      color: 'from-secondary-500 to-secondary-600',
    },
    {
      label: '存储占用',
      value: formattedSize,
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
        </svg>
      ),
      color: 'from-accent-500 to-accent-600',
    },
    {
      label: '标签数量',
      value: (stats.total_tags ?? 0).toString(),
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
        </svg>
      ),
      color: 'from-emerald-500 to-emerald-600',
    },
  ]

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <div key={card.label} className="card dark:card-dark p-4 hover:shadow-lg transition-shadow duration-300">
          <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${card.color} flex items-center justify-center text-white mb-3`}>
            {card.icon}
          </div>
          <p className="text-2xl font-bold text-gray-900 dark:text-dark-text">{card.value}</p>
          <p className="text-sm text-gray-500 dark:text-dark-muted mt-0.5">{card.label}</p>
        </div>
      ))}
    </div>
  )
}
