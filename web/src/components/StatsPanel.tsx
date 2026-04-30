import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

interface Stats {
  total_photos: number
  total_size: number
  total_tags: number
  total_faces?: number
  total_persons?: number
  tag_breakdown?: Record<string, number>
}

interface StatsPanelProps {
  selectedDir?: string
}

function getTagSizeClass(index: number, total: number): string {
  const percentile = index / Math.max(1, total - 1)
  if (percentile <= 0.1) return 'text-2xl font-bold px-4 py-2'
  if (percentile <= 0.3) return 'text-xl font-semibold px-3.5 py-1.5'
  if (percentile <= 0.6) return 'text-base font-medium px-3 py-1.5'
  return 'text-sm font-normal px-2.5 py-1'
}

export default function StatsPanel({ selectedDir }: StatsPanelProps) {
  const navigate = useNavigate()
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [showTagsModal, setShowTagsModal] = useState(false)
  const [faceStats, setFaceStats] = useState({ total_faces: 0, total_persons: 0 })

  useEffect(() => {
    // 获取人脸统计
    fetch('/api/v1/faces/stats')
      .then(res => {
        if (!res.ok) throw new Error('faces stats failed')
        return res.json()
      })
      .then(data => {
        setFaceStats({
          total_faces: data.total_faces || 0,
          total_persons: data.total_persons || 0
        })
      })
      .catch(() => {
        // 静默失败，不影响主统计
      })
  }, [])

  useEffect(() => {
    setLoading(true)
    const url = selectedDir
      ? `/api/v1/stats?dir=${encodeURIComponent(selectedDir)}`
      : '/api/v1/stats'
    fetch(url)
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
  }, [selectedDir])

  const tagBreakdown = stats?.tag_breakdown
  const sortedTags = tagBreakdown
    ? Object.entries(tagBreakdown).sort((a, b) => b[1] - a[1])
    : []
  const maxCount = sortedTags[0]?.[1] ?? 1
  const rawSize = stats?.total_size ?? 0
  const formattedSize = rawSize > 1073741824
    ? `${(rawSize / 1073741824).toFixed(1)} GB`
    : rawSize > 1048576
      ? `${(rawSize / 1048576).toFixed(1)} MB`
      : `${(rawSize / 1024).toFixed(0)} KB`

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
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
      clickable: false,
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
      clickable: false,
    },
    {
      label: '标签数量',
      value: (stats.total_tags ?? 0).toLocaleString(),
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
        </svg>
      ),
      color: 'from-emerald-500 to-emerald-600',
      clickable: true,
    },
    {
      label: '发现人物',
      value: faceStats.total_persons > 0 ? `${faceStats.total_persons}人` : '0',
      subValue: faceStats.total_faces > 0 ? `${faceStats.total_faces}张人脸` : undefined,
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.592-2.641m-5.807 2.641l-.001.109M6.375 16.813c0-1.113.285-2.16.786-3.07m0 0a6.375 6.375 0 0111.592-2.641m-5.807 2.641l-.001.109M12 12a3 3 0 11-6 0 3 3 0 016 0zm6 0a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
      color: 'from-rose-500 to-rose-600',
      clickable: true,
      onClick: () => navigate('/persons'),
    },
  ]

  const handleTagClick = (tag: string) => {
    setShowTagsModal(false)
    navigate(`/search?q=${encodeURIComponent(tag)}`)
  }

  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map((card) => (
          <div
            key={card.label}
            onClick={() => {
              if (card.onClick) {
                card.onClick()
              } else if (card.clickable) {
                setShowTagsModal(true)
              }
            }}
            className={`card dark:card-dark p-4 hover:shadow-lg transition-all duration-300 ${
              card.clickable ? 'cursor-pointer hover:scale-[1.02]' : ''
            }`}
          >
            <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${card.color} flex items-center justify-center text-white mb-3`}>
              {card.icon}
            </div>
            <p className="text-2xl font-bold text-gray-900 dark:text-dark-text">{card.value}</p>
            {card.subValue && (
              <p className="text-xs text-gray-500 dark:text-dark-muted mt-0.5">{card.subValue}</p>
            )}
            <p className="text-sm text-gray-500 dark:text-dark-muted mt-0.5">{card.label}</p>
            {card.clickable && !card.onClick && (
              <p className="text-xs text-emerald-600 dark:text-emerald-400 mt-2 font-medium">点击查看标签词云 →</p>
            )}
            {card.onClick && (
              <p className="text-xs text-rose-600 dark:text-rose-400 mt-2 font-medium">点击查看人物相册 →</p>
            )}
          </div>
        ))}
      </div>

      {showTagsModal && (
        <div
          className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-4 animate-fade-in"
          onClick={() => setShowTagsModal(false)}
        >
          <div
            className="bg-white dark:bg-dark-card rounded-2xl shadow-2xl border border-gray-100 dark:border-dark-border w-full max-w-2xl max-h-[80vh] flex flex-col animate-slide-up"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b border-gray-100 dark:border-dark-border">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text">标签词云</h3>
              <button
                onClick={() => setShowTagsModal(false)}
                className="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-dark-text hover:bg-gray-100 dark:hover:bg-dark-surface transition-colors"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="p-6 overflow-y-auto">
              {sortedTags.length === 0 ? (
                <p className="text-center text-gray-500 dark:text-dark-muted py-8">暂无标签数据</p>
              ) : (
                <div className="flex flex-wrap items-center justify-center gap-3">
                  {sortedTags.map(([tag, count], index) => {
                    const sizeClass = getTagSizeClass(index, sortedTags.length)
                    const opacity = 0.4 + (0.6 * (count / maxCount))
                    return (
                      <button
                        key={tag}
                        onClick={() => handleTagClick(tag)}
                        className={`rounded-full transition-all duration-200 hover:scale-105 ${sizeClass} ${
                          index % 5 === 0
                            ? 'bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-400 hover:bg-primary-100 dark:hover:bg-primary-500/20'
                            : index % 5 === 1
                            ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-500/20'
                            : index % 5 === 2
                            ? 'bg-accent-50 text-accent-600 dark:bg-accent-500/10 dark:text-accent-400 hover:bg-accent-100 dark:hover:bg-accent-500/20'
                            : index % 5 === 3
                            ? 'bg-secondary-50 text-secondary-600 dark:bg-secondary-500/10 dark:text-secondary-400 hover:bg-secondary-100 dark:hover:bg-secondary-500/20'
                            : 'bg-gray-100 text-gray-600 dark:bg-dark-surface dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border'
                        }`}
                        style={{ opacity }}
                        title={`${tag} (${count} 张)`}
                      >
                        {tag}
                        <span className="ml-1.5 text-[0.65em] opacity-70 font-normal">{count}</span>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>

            <div className="p-4 border-t border-gray-100 dark:border-dark-border flex justify-between items-center">
              <p className="text-sm text-gray-500 dark:text-dark-muted">
                共 <span className="font-medium text-gray-900 dark:text-dark-text">{sortedTags.length}</span> 个标签
              </p>
              <button
                onClick={() => setShowTagsModal(false)}
                className="px-4 py-2 rounded-lg bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border transition-colors text-sm font-medium"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
