import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import SearchBar from '../components/SearchBar'
import PhotoGrid from '../components/PhotoGrid'

interface SearchResult {
  PhotoID: string
  Score: number
  PhotoPath: string
}

type Period = 'all' | 'week' | 'month' | 'year'

const PERIOD_LABELS: Record<Period, string> = {
  all: '全部',
  week: '本周',
  month: '本月',
  year: '本年',
}

const HOT_TAGS = ['人像', '风景', '美食', '旅行', '宠物', '建筑']

interface SearchProps {
  selectedDir: string
}

export default function Search({ selectedDir }: SearchProps) {
  const [searchParams, setSearchParams] = useSearchParams()
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [period, setPeriod] = useState<Period>((searchParams.get('period') as Period) || 'all')
  const [selectedTags, setSelectedTags] = useState<string[]>(
    searchParams.get('tags')?.split(',').filter(Boolean) || []
  )
  const [isSearching, setIsSearching] = useState(false)
  const [results, setResults] = useState<SearchResult[]>([])
  const [total, setTotal] = useState(0)

  useEffect(() => {
    const q = searchParams.get('q')
    const p = (searchParams.get('period') as Period) || 'all'
    const t = searchParams.get('tags')?.split(',').filter(Boolean) || []
    if (q) {
      setSearchQuery(q)
      setPeriod(p)
      setSelectedTags(t)
      performSearch(q, p, t, selectedDir)
    }
  }, [searchParams, selectedDir])

  const performSearch = async (query: string, searchPeriod: Period, tags: string[], dir: string) => {
    setIsSearching(true)
    try {
      const params = new URLSearchParams()
      params.append('q', query)
      params.append('period', searchPeriod)
      if (tags.length > 0) {
        params.append('tags', tags.join(','))
      }
      if (dir) {
        params.append('dir', dir)
      }
      const response = await fetch(`/api/v1/search?${params.toString()}`)
      const data = await response.json()
      setResults(data.results || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Search failed:', error)
      setResults([])
      setTotal(0)
    } finally {
      setIsSearching(false)
    }
  }

  const updateSearchParams = (q: string, p: Period, tags: string[]) => {
    const params: Record<string, string> = { q }
    if (p !== 'all') params.period = p
    if (tags.length > 0) params.tags = tags.join(',')
    setSearchParams(params)
  }

  const handleSearch = (query: string) => {
    updateSearchParams(query, period, selectedTags)
  }

  const handlePeriodChange = (p: Period) => {
    setPeriod(p)
    if (searchQuery) {
      updateSearchParams(searchQuery, p, selectedTags)
    }
  }

  const toggleTag = (tag: string) => {
    const next = selectedTags.includes(tag)
      ? selectedTags.filter((t) => t !== tag)
      : [...selectedTags, tag]
    setSelectedTags(next)
    if (searchQuery) {
      updateSearchParams(searchQuery, period, next)
    }
  }

  const hasFilters = period !== 'all' || selectedTags.length > 0

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Search Header */}
      <div className="mb-8">
        <div className="max-w-3xl mx-auto">
          <SearchBar onSearch={handleSearch} placeholder="搜索照片..." />
        </div>

        {/* Filters */}
        {searchQuery && (
          <div className="mt-4 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-gray-400 dark:text-dark-muted mr-1">时间:</span>
              {(Object.keys(PERIOD_LABELS) as Period[]).map((p) => (
                <button
                  key={p}
                  onClick={() => handlePeriodChange(p)}
                  className={`px-3 py-1.5 text-xs rounded-full font-medium transition-colors ${
                    period === p
                      ? 'bg-primary-500 text-white'
                      : 'bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card'
                  }`}
                >
                  {PERIOD_LABELS[p]}
                </button>
              ))}
              {selectedDir && (
                <span className="ml-auto px-2 py-1 text-[10px] rounded bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">
                  仅搜索: {selectedDir.split('/').pop()}
                </span>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-gray-400 dark:text-dark-muted mr-1">标签:</span>
              {HOT_TAGS.map((tag) => (
                <button
                  key={tag}
                  onClick={() => toggleTag(tag)}
                  className={`px-3 py-1.5 text-xs rounded-full font-medium transition-colors ${
                    selectedTags.includes(tag)
                      ? 'bg-emerald-500 text-white'
                      : 'bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card'
                  }`}
                >
                  {tag}
                </button>
              ))}
              {hasFilters && (
                <button
                  onClick={() => {
                    setPeriod('all')
                    setSelectedTags([])
                    if (searchQuery) updateSearchParams(searchQuery, 'all', [])
                  }}
                  className="px-3 py-1.5 text-xs rounded-full text-gray-400 hover:text-gray-600 dark:hover:text-dark-text underline"
                >
                  清除筛选
                </button>
              )}
            </div>

            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-display font-semibold text-gray-900 dark:text-dark-text">
                  搜索结果："{searchQuery}"
                </h2>
                <p className="text-sm text-gray-500 dark:text-dark-muted mt-1">
                  {isSearching ? (
                    <span className="flex items-center gap-2">
                      <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      正在搜索...
                    </span>
                  ) : `找到 ${total} 张相关照片`}
                </p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Photo Grid */}
      <PhotoGrid
        loading={isSearching}
        dir={selectedDir}
        photos={results.map((r) => {
          const photoPath = (r as any).path || r.PhotoPath
          return {
            id: r.PhotoID,
            url: `/api/v1/photos/file?path=${encodeURIComponent(photoPath)}`,
            thumbnail: `/api/v1/photos/file?path=${encodeURIComponent(photoPath)}`,
            title: (r as any).name || r.PhotoID,
            date: (r as any).date_time || '',
            tags: (r as any).tags || ['搜索结果'],
            width: (r as any).width || 0,
            height: (r as any).height || 0,
            cameraMake: (r as any).camera_make || '',
            cameraModel: (r as any).camera_model || '',
          }
        })}
      />
    </main>
  )
}
