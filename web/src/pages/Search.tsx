import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import SearchBar from '../components/SearchBar'
import PhotoGrid from '../components/PhotoGrid'

interface SearchResult {
  PhotoID: string
  Score: number
  PhotoPath: string
  path?: string
  name?: string
  date_time?: string
  tags?: string[]
  width?: number
  height?: number
  camera_make?: string
  camera_model?: string
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
  const [page, setPage] = useState(1)
  const PAGE_SIZE = 40

  useEffect(() => {
    const q = searchParams.get('q')
    const p = (searchParams.get('period') as Period) || 'all'
    const t = searchParams.get('tags')?.split(',').filter(Boolean) || []
    if (q) {
      setSearchQuery(q)
      setPeriod(p)
      setSelectedTags(t)
      setPage(1)
      performSearch(q, p, t, selectedDir, 1)
    }
  }, [searchParams, selectedDir])

  const performSearch = async (query: string, searchPeriod: Period, tags: string[], dir: string, targetPage: number, append = false) => {
    setIsSearching(!append)
    try {
      const params = new URLSearchParams()
      params.append('q', query)
      params.append('period', searchPeriod)
      params.append('page', String(targetPage))
      params.append('page_size', String(PAGE_SIZE))
      if (tags.length > 0) params.append('tags', tags.join(','))
      if (dir) params.append('dir', dir)

      const response = await fetch(`/api/v1/search?${params.toString()}`)
      const data = await response.json()
      const newResults = data.results || []
      setResults(prev => append ? [...prev, ...newResults] : newResults)
      setTotal(data.total || 0)
      setPage(targetPage)
    } catch (error) {
      console.error('Search failed:', error)
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
    if (searchQuery) updateSearchParams(searchQuery, p, selectedTags)
  }

  const toggleTag = (tag: string) => {
    const next = selectedTags.includes(tag)
      ? selectedTags.filter(t => t !== tag)
      : [...selectedTags, tag]
    setSelectedTags(next)
    if (searchQuery) updateSearchParams(searchQuery, period, next)
  }

  const hasMore = results.length < total

  const handleLoadMore = useCallback(() => {
    if (!hasMore || isSearching) return
    performSearch(searchQuery, period, selectedTags, selectedDir, page + 1, true)
  }, [hasMore, isSearching, searchQuery, period, selectedTags, selectedDir, page])

  const hasFilters = period !== 'all' || selectedTags.length > 0

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-6">
        <div className="max-w-3xl mx-auto">
          <SearchBar onSearch={handleSearch} placeholder="搜索照片..." />
        </div>

        {searchQuery && (
          <div className="mt-4 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-gray-400 mr-1">时间:</span>
              {(Object.keys(PERIOD_LABELS) as Period[]).map(p => (
                <button
                  key={p}
                  onClick={() => handlePeriodChange(p)}
                  className={`px-3 py-1.5 text-xs rounded-full font-medium transition-colors ${
                    period === p
                      ? 'bg-purple-500 text-white'
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
              <span className="text-xs text-gray-400 mr-1">标签:</span>
              {HOT_TAGS.map(tag => (
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
                  onClick={() => { setPeriod('all'); setSelectedTags([]); if (searchQuery) updateSearchParams(searchQuery, 'all', []) }}
                  className="px-3 py-1.5 text-xs rounded-full text-gray-400 hover:text-gray-600 dark:hover:text-dark-text underline"
                >
                  清除筛选
                </button>
              )}
            </div>

            <div>
              <h2 className="text-lg font-display font-semibold text-gray-900 dark:text-white">
                「{searchQuery}」
              </h2>
              <p className="text-sm text-gray-500 dark:text-dark-muted mt-0.5">
                {total > 0 ? `找到 ${total} 张照片` : '搜索中…'}
              </p>
            </div>
          </div>
        )}
      </div>

      <PhotoGrid
        loading={isSearching && results.length === 0}
        dir={selectedDir}
        hasMore={hasMore}
        onLoadMore={handleLoadMore}
        photos={results.map(r => {
          const photoPath = r.path || r.PhotoPath
          return {
            id: r.PhotoID,
            url: `/api/v1/photos/file?path=${encodeURIComponent(photoPath)}`,
            thumbnail: `/api/v1/photos/file?path=${encodeURIComponent(photoPath)}`,
            title: r.name || r.PhotoID,
            date: r.date_time || '',
            tags: r.tags || [],
            width: r.width || 0,
            height: r.height || 0,
            cameraMake: r.camera_make || '',
            cameraModel: r.camera_model || '',
          }
        })}
      />
    </main>
  )
}
