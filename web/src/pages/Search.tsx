import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import SearchBar from '../components/SearchBar'
import PhotoGrid from '../components/PhotoGrid'

interface SearchResult {
  PhotoID: string
  Score: number
  PhotoPath: string
}

export default function Search() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [isSearching, setIsSearching] = useState(false)
  const [results, setResults] = useState<SearchResult[]>([])
  const [total, setTotal] = useState(0)

  useEffect(() => {
    const q = searchParams.get('q')
    if (q) {
      setSearchQuery(q)
      performSearch(q)
    }
  }, [searchParams])

  const performSearch = async (query: string) => {
    setIsSearching(true)
    try {
      const response = await fetch(`/api/v1/search?q=${encodeURIComponent(query)}`)
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

  const handleSearch = (query: string) => {
    setSearchQuery(query)
    setSearchParams({ q: query })
  }

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Search Header */}
      <div className="mb-8">
        <div className="max-w-3xl mx-auto">
          <SearchBar onSearch={handleSearch} placeholder="搜索照片..." />
        </div>

        {/* Search Info */}
        {searchQuery && (
          <div className="mt-4 flex items-center justify-between">
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

            {/* Filter Chips */}
            <div className="hidden md:flex items-center space-x-2">
              <button className="px-3 py-1.5 text-xs rounded-full bg-primary-500 text-white font-medium">
                全部
              </button>
              <button className="px-3 py-1.5 text-xs rounded-full bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card transition-colors">
                本周
              </button>
              <button className="px-3 py-1.5 text-xs rounded-full bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card transition-colors">
                本月
              </button>
              <button className="px-3 py-1.5 text-xs rounded-full bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card transition-colors">
                本年
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Photo Grid */}
      <PhotoGrid
        loading={isSearching}
        photos={results.map((r) => {
          const photoPath = (r as any).path || r.PhotoPath
          return {
            id: r.PhotoID,
            url: `/api/v1/photo?path=${encodeURIComponent(photoPath)}`,
            thumbnail: `/api/v1/photo?path=${encodeURIComponent(photoPath)}`,
            title: (r as any).name || r.PhotoID,
            date: (r as any).date_time || '',
            tags: (r as any).tags || ['搜索结果'],
            width: 400,
            height: 300,
          }
        })}
      />
    </main>
  )
}
